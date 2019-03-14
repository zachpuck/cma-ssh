#!/bin/bash

DEBUG=${DEBUG:-1}

[[ $DEBUG == 1 ]] && \
  {
    set -o xtrace
  }

set -o errexit
set -o nounset
set -o pipefail

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

readonly CLIENT_KUBECONFIG="$CLUSTER_NAME-kubeconfig.yaml"

get_kubeconfig()
{
  "${__dir}/get-kubeconfig" > "${CLIENT_KUBECONFIG}"
}

test_provisioning()
{
  if [[ $("${__dir}"/cma-create | jq -Mr '.status') == PROVISIONING ]]; then
    echo "Cluster is PROVISIONING"
  else
    echo "Cluster is NOT PROVISIONING"
    return 1
  fi
  return 0
}

test_running()
{
  # wait up to 20 minutes for cluster RUNNING
  for tries in $(seq 1 120); do

    if [[ $("${__dir}"/cma-get | jq -Mr '.cluster.status') == RUNNING ]]; then
      echo "Cluster is RUNNING"
      runningstatus="PASS"
      echo "elapsed seconds=$(( 10 * $tries ))"
      break
    else
      echo "Cluster is NOT RUNNING"
    fi
    sleep 10
  done

  if [ -z ${runningstatus+x} ]; then
    echo "Timed out waiting for RUNNING status"
    return 1
  fi
  return 0
}

test_ready()
{
  local c
  c=0

  get_kubeconfig

  while [[ $c -le 120 ]]; do
    if ! nodes=$(kubectl get nodes --kubeconfig "$CLIENT_KUBECONFIG"); then
      return 1
    fi

    echo "$nodes"

    # check for not ready
    if echo "$nodes" | grep -qo NotReady; then
      echo "Node(s) NotReady"
    fi

    if echo "$nodes" | grep -qo SchedulingDisabled; then
      echo "Node(s) SchedulingDisabled"
    fi

    if echo "$nodes" | grep -qo Ready; then
      echo "Node(s) Ready"
      rm "$CLIENT_KUBECONFIG"
      return 0
    fi

    sleep 1
    ((c++))
  done

  rm "$CLIENT_KUBECONFIG"
  return 1
}

test_delete()
{
  jqfilter='if ( .status == null ) then .cluster.status else .status end'

  if [[ $( "${__dir}"/cma-delete | jq -Mr "$jqfilter") != "Deleted" ]]; then
    echo >&2 "REST call to API failed to get valid response 'Deleted'"
  fi

  # wait up to 20 minutes for cluster delete complete
  for tries in $(seq 1 120); do
    if [[ $("${__dir}"/cma-get | jq -Mr "$jqfilter") == STOPPING ]]; then
      echo "Cluster DELETE is COMPLETE"
      deletedstatus="PASS"
      echo "elapsed seconds=$(( 10 * "$tries" ))"
      break
    else
      echo "Cluster DELETE is NOT COMPLETE"
    fi
    sleep 10
  done

  if [ -z ${deletedstatus+x} ]; then
    echo "Timed out waiting for DELETE to finish"
    return 1
  fi
  return 0
}


main()
{
  fullstatus="PASSED"

  # test create is provisioning
  if ! test_provisioning; then
     echo "test_provisioning FAILED"
  else
     echo "test_provisioning PASSED"
  fi

  if ! test_running; then
    echo "test_running FAILED"
    fullstatus="FAILED"
  else
    echo "test_running PASSED"
  fi

  if ! test_ready; then
    echo "test_ready FAILED"
    fullstatus="FAILED"
  else
    echo "test_ready PASSED"
  fi

  if ! test_delete; then
    echo "test_delete FAILED"
    fullstatus="FAILED"
  else
    echo "test_delete PASSED"
  fi

  echo "full-test $fullstatus"
  if [ "$fullstatus" == "FAILED" ]; then
    exit 1
  fi

  exit 0
}

main
