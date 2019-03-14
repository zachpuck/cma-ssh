# Purpose

This e2e tests CMA using the SSH provider. These tests demonstrate base end to end functionality of the cma-ssh helper of the Cluster Manager API. The use of `cma-ssh` in this test is mildly inconsequential as the primary successful test story for this test is to make sure that CMA works end-to-end.

# Context

This set of tests using the Cluster Manager API pipeline through CMA SSH, which will install a managed cluster on pre-provisioned hosts. In order for this specific set of tests to work, you *must* have at least two IP addresses (machines, vms, etc) running BEFORE these tests can be performed.

# How to run

1.  populate the environment with required VARs (defaults are provided)  See all of the environmentals needed in full-test.sh
2.  execute `full-test.sh`

## Sequence of the tests
(FIX UP - WIP)
1.  create a client cluster via a parent CMA-SSH helper:
    `create-cluster.sh`
2.  get the kubeconfig for the client cluster from the parent cluster
    K8S API `get-kubeconfig()` in `full-test.sh`
3.  create a simple system in the client cluster (using nginx-ingress)
4.  verify the simple system functions
5.  tear down the client cluster: `delete-cluster.sh`
