#!/usr/bin/env sh

echo "create cluster"
kind create cluster --name $name

cat ~/.kube/kind-config-$name > /repo/kind-kubeConfig.yaml
cat ~/.kube/kind-config-$name | sed s/localhost/$HOST_MACHINE_IP/ > /kubeConfig
