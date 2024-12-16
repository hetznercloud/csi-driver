#!/usr/bin/env bash
set -e -o pipefail

if [ "$DEBUG" != "" ]; then
  set -x
fi

PV_NAME="$1"

# Prepare directory
DIR="./hcloud-csi-fix-topology/${PV_NAME}"
echo "[INFO] Creating a new directory to backup objects: ${DIR}"
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
  if ! command -v "$cmd" &> /dev/null; then
    write_log "[ERR] For the script to run successfully, \"${cmd}\" is required, but it could not be found. Please make sure it is installed."
    exit
  fi
}

verify_installed kubectl
verify_installed hcloud

# [kubectl] Get PersistentVolume (PV) and verify it fulfills criteria
PV_FILE_ORIG="${DIR}/persistentvolume.orig.json"

kubectl get persistentvolume "${PV_NAME}" -o=json > "$PV_FILE_ORIG"
mapfile -t PV_INFO < <(
    kubectl get persistentvolume "${PV_NAME}" \
      -o=jsonpath='{.metadata.annotations.pv\.kubernetes\.io\/provisioned-by} {.spec.nodeAffinity.required.nodeSelectorTerms[*].matchExpressions[*].key} {.spec.csi.volumeHandle}'
)
PV_PROVISIONED_BY="${PV_INFO[0]}"
PV_TOPOLOGY_LABEL="${PV_INFO[1]}"
PV_VOLUME_ID="${PV_INFO[2]}"

if [ "${PV_PROVISIONED_BY}" != "csi.hetzner.cloud" ]; then
  write_log "[ERR] PersistentVolume with name \"${PV_NAME}\" was not provisioned by hcloud-csi-driver."
  exit 1
fi

if [ "${PV_TOPOLOGY_LABEL}" != "topology.kubernetes.io/region" ]; then
  write_log "[ERR] PersistentVolume with name \"${PV_NAME}\" does not use the invalid topology label."
  exit 1
fi

# [kubectl] Verify that no volume attachment exists
ATTACHMENTS=$(kubectl get volumeattachment -o jsonpath="{.items[?(@.spec.source.persistentVolumeName==\"${PV_NAME}\")].metadata.name}")
if [ "${ATTACHMENTS}" != "" ]; then
  write_log "[ERR] PersistentVolume with name \"${PV_NAME}\" is still attached according to kubernetes VolumeAttachment: ${ATTACHMENTS}"
  exit 1
fi

# [hcloud] Get Volume
hcloud volume describe "${PV_VOLUME_ID}" -o=json > "${DIR}"/volume.orig.json
mapfile -t VOLUME_INFO < <(hcloud volume describe "${PV_VOLUME_ID}" -o=format='{{.Protection.Delete}} {{if .Server }}{{.Server.ID}}{{end}}')

VOLUME_DELETION_PROTECTION="${VOLUME_INFO[0]}"
VOLUME_SERVER="${VOLUME_INFO[1]}"

# [hcloud] Verify that the Volume is not assigned to a server
if [ "${VOLUME_SERVER}" != "" ]; then
  write_log "[ERR] Hetzner Cloud Volume with ID \"${PV_VOLUME_ID}\" is still attached to server \"${VOLUME_SERVER}\" according to Hetzner Cloud API."
  exit 1
fi

# [hcloud] Enable deletion protection
write_log "[INFO] Current state of Volume deletion protection: ${VOLUME_DELETION_PROTECTION}"

if [ "${VOLUME_DELETION_PROTECTION}" != "true" ]; then
  write_log "[INFO] Enabling Volume deletion protection"
  hcloud volume enable-protection "${PV_VOLUME_ID}" delete
fi

# [kubectl] Remove finalizers
write_log "[INFO] Removing finalizers from PersistentVolume"
kubectl patch persistentvolume "${PV_NAME}" --type=json -p='[{"op":"replace", "path": "/metadata/finalizers", "value": []}]'

# Prepare PersistentVolume JSON
PV_FILE_FIXED="${DIR}/persistentvolume.fixed.json"
kubectl patch \
  --dry-run=client \
  --filename="$PV_FILE_ORIG" \
  --type=json \
  --patch='[{"op":"replace", "path": "/spec/nodeAffinity/required/nodeSelectorTerms/0/matchExpressions/0/key", "value": "csi.hetzner.cloud/location"}]' \
  --output=yaml > "${PV_FILE_FIXED}"

# [kubectl] Delete Persistent Volume
write_log "[INFO] Deleting current PersistentVolume"
kubectl delete persistentvolume "${PV_NAME}" &
# The pv-protection finalizer is added right back. For the deletion to work,
# we need to remove it again.
kubectl patch persistentvolume "${PV_NAME}" --type=json -p='[{"op":"replace", "path": "/metadata/finalizers", "value": []}]'

write_log "[INFO] Waiting for deletion to finish"
kubectl wait --for=delete persistentvolume "${PV_NAME}" --timeout=10s

# [kubectl] Create new Persistent Volume
write_log "[INFO] Creating new PersistentVolume"
kubectl create --filename="${PV_FILE_FIXED}"

# [hcloud] Disable deletion protection (if previously enabled)
if [ "${VOLUME_DELETION_PROTECTION}" != "true" ]; then
  write_log "[INFO] Disabling Volume deletion protection which was added for migration"
  hcloud volume disable-protection "${PV_VOLUME_ID}" delete
fi
