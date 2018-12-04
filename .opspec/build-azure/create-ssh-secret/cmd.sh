#!/usr/bin/env sh

echo "create namespace"
if [ "$(kubectl get namespace $namespace --no-headers=true -o custom-columns=:metadata.name)" != "" ]
then
  echo "found exiting namespace"
else
    echo "creating namespace"
    kubectl create namespace $namespace
fi
