#!/usr/bin/env bash
set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Redirect all stdout to stderr.
{
  if ! hcloud version >/dev/null; then
    echo 'ERROR: `hcloud` CLI not found, please install it and make it available on your $PATH'
    exit 1
  fi

  if ! k3sup version >/dev/null; then
    echo 'ERROR: `k3sup` not found, please install it and make it available on your $PATH'
    exit 1
  fi

  if [[ "${HCLOUD_TOKEN:-}" == "" ]]; then
    echo 'ERROR: please set $HCLOUD_TOKEN'
    exit 1
  fi

  # We run a lot of subshells below for speed. If any encounter an error, we shut down the whole process group, pronto.
  function error() {
    echo 'Onoes, something went wrong! :( The output above might have some clues.'
    kill 0
  }

  trap error ERR

  image_name=${IMAGE_NAME:-ubuntu-20.04}
  instance_count=${INSTANCES:-1}
  instance_type=${INSTANCE_TYPE:-cpx11}
  location=${LOCATION:-fsn1}
  ssh_keys=${SSH_KEYS:-}
  channel=${K3S_CHANNEL:-stable}

  scope="${SCOPE:-dev}"
  label="managedby=hack,scope=$scope"
  ssh_private_key="$SCRIPT_DIR/.ssh-$scope"
  k3s_opts=${K3S_OPTS:-"--kubelet-arg cloud-provider=external"}
  k3s_server_opts=${K3S_SERVER_OPTS:-"--disable-cloud-controller --cluster-cidr 10.244.0.0/16"}

  scope_name=csi-driver-${scope}

  export KUBECONFIG="$SCRIPT_DIR/.kubeconfig-$scope"

  ssh_command="ssh -i $ssh_private_key -o StrictHostKeyChecking=off -o BatchMode=yes -o ConnectTimeout=5"

  # Generate SSH keys and upload publkey to Hetzner Cloud.
  ( trap error ERR
    [[ ! -f $ssh_private_key ]] && ssh-keygen -t ed25519 -f $ssh_private_key -C '' -N ''
    [[ ! -f $ssh_private_key.pub ]] && ssh-keygen -y -f $ssh_private_key > $ssh_private_key.pub
    if ! hcloud ssh-key describe $scope_name >/dev/null 2>&1; then
      hcloud ssh-key create --label $label --name $scope_name --public-key-from-file $ssh_private_key.pub
    fi
  ) &

  for num in $(seq $instance_count); do
    # Create server and initialize Kubernetes on it with k3sup.
    ( trap error ERR

      server_name="$scope_name-$num"

      # Maybe cluster is already up and node is already there.
      if kubectl get node $server_name >/dev/null 2>&1; then
        exit 0
      fi

      ip=$(hcloud server ip $server_name 2>/dev/null || true)

      if [[ -z "${ip:-}" ]]; then
        # Wait for SSH key
        until hcloud ssh-key describe $scope_name >/dev/null 2>&1; do sleep 1; done

        createcmd="hcloud server create --image $image_name --label $label --location $location --name $server_name --ssh-key=$scope_name --type $instance_type"
        for key in $ssh_keys; do
          createcmd+=" --ssh-key $key"
        done
        $createcmd
        ip=$(hcloud server ip $server_name)
      fi

      # Wait for SSH.
      until [ "$($ssh_command root@$ip echo ok 2>/dev/null)" = "ok" ]; do
        sleep 1
      done

      if [[ "$num" == "1" ]]; then
        # First node is control plane.
        k3sup install --print-config=false --ip $ip --k3s-channel $channel --k3s-extra-args "${k3s_server_opts} ${k3s_opts}" --local-path $KUBECONFIG --ssh-key $ssh_private_key
      else
        # All subsequent nodes are initialized as workers.

        # Can't go any further until control plane has bootstrapped a bit though.
        until $ssh_command root@$(hcloud server ip $scope_name-1 || true) stat /etc/rancher/node/password >/dev/null 2>&1; do
          sleep 1
        done

        k3sup join --server-ip $(hcloud server ip $scope_name-1) --ip $ip --k3s-channel $channel --k3s-extra-args "${k3s_opts}" --ssh-key $ssh_private_key
      fi
    ) &

    # Wait for this node to show up in the cluster.
    ( trap error ERR; set +x
      until kubectl wait --for=condition=Ready node/$scope_name-$num >/dev/null 2>&1; do sleep 1; done
      echo $scope_name-$num is up and in cluster
    ) &
  done

  ( trap error ERR
    # Control plane init tasks.
    # This is running in parallel with the server init, above.

    # Wait for control plane to look alive.
    until kubectl get nodes >/dev/null 2>&1; do sleep 1; done;

    # Install flannel.
    ( trap error ERR
      if ! kubectl get -n kube-system ds/kube-flannel-ds >/dev/null 2>&1; then
        kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
        kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{"op":"add","path":"/spec/template/spec/tolerations/-","value":{"key":"node.cloudprovider.kubernetes.io/uninitialized","value":"true","effect":"NoSchedule"}}]'
      fi) &

    # Create HCLOUD_TOKEN Secret for hcloud-cloud-controller-manager.
    ( trap error ERR
      if ! kubectl -n kube_system get secret hcloud >/dev/null 2>&1; then
        kubectl -n kube-system create secret generic hcloud --from-literal="token=$HCLOUD_TOKEN"
      fi) &

    # Install hcloud-cloud-controller-manager.
    ( trap error ERR
      if ! kubectl get -n kube-system deploy/hcloud-cloud-controller-manager >/dev/null 2>&1; then
        kubectl apply -f https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/ccm.yaml
      fi) &
    wait
  ) &

  wait
  echo "Success - cluster fully initialized and ready, why not see for yourself?"
  echo '$ kubectl get nodes'
  kubectl get nodes
} >&2

echo "export KUBECONFIG=$KUBECONFIG"
