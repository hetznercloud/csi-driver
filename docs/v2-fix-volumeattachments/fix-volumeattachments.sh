#!/usr/bin/env bash
set -e -o pipefail

if [ "$DEBUG" != "" ];
then
  set -x
fi

DIR="./hcloud-csi-fix-volumeattachments"
echo "[INFO] Creating a new directory to write logs: ${DIR}"
mkdir --parents "${DIR}"

# Logging utility
LOG_FILE="${DIR}/logs.txt"
write_log() {
  echo "$1"
  echo "$1" >> "${LOG_FILE}"
}

# Verify dependencies
verify_installed() {
  cmd="$1"
  if ! command -v "$cmd" &> /dev/null
  then
    write_log "[ERR] For the script to run successfully, \"${cmd}\" is required, but it could not be found. Please make sure it is installed."
    exit
  fi
}

verify_installed kubectl
verify_installed grep

VOLUME_ATTACHMENTS=$(
  kubectl get volumeattachment \
    -o custom-columns=NAME:.metadata.name,ATTACHER:.spec.attacher,DEVICEPATH:.status.attachmentMetadata.devicePath \
  | { grep -E 'csi\.hetzner\.cloud.*<none>' --color=never || true; } \
  | cut --fields=1 --delimiter=' '
)

if [[ -z "$VOLUME_ATTACHMENTS" ]]; then
  write_log "[INFO] No affected VolumeAttachments found, exiting."
  exit 0
fi

for VOLUME_ATTACHMENT in "$VOLUME_ATTACHMENTS"; do
  write_log "[INFO] Processing VolumeAttachment $VOLUME_ATTACHMENT"


  PV_NAME=$(
    kubectl get volumeattachment \
      -o=jsonpath="{.spec.source.persistentVolumeName}" \
      "$VOLUME_ATTACHMENT"
  )

  VOLUME_ID=$(
    kubectl get persistentvolume \
      -o=jsonpath="{.spec.csi.volumeHandle}" \
      "$PV_NAME"
  )

  write_log "[INFO] VolumeAttachment $VOLUME_ATTACHMENT uses volume $VOLUME_ID"

  DEVICE_PATH="/dev/disk/by-id/scsi-0HC_Volume_${VOLUME_ID}"

  kubectl patch volumeattachment \
    --subresource=status \
    --type=strategic \
    -p "{\"status\":{\"attachmentMetadata\": {\"devicePath\":\"${DEVICE_PATH}\"}}}" \
    "$VOLUME_ATTACHMENT"

  write_log "[INFO] Patched VolumeAttachment $VOLUME_ATTACHMENT"
done

write_log "[INFO] Finished processing all VolumeAttachments!"
