cma-ssh OPCTL README

# opctl

requires [opctl](https://opctl.io/docs/getting-started/opctl.html) installed

# Working with Azure

1. create the ssh key pair (requires rsa and 2048 bit)
    **no password**
    `ssh-keygen -t rsa -b 2048 -f id_rsa`
    
2. create `args.yml` file
    ```
    touch .opspec/args.yml
    ```
    paste provided inputs in args file.

3. run `opctl run build-azure` and wait for completion (about 10 to 15 minutes on first run)
    *this can be run multiple times
4. to get kubeconfig for cluster:
    - install azure cli: `brew install azure-cli`
    - login to azure via cli: `az login`
    - get kubeconfig from aks cluster: `az aks get-credentials -n <name> -g <name>-group` *replace with name from args.yml

5. example helm install: `helm install deployments/helm/cma-ssh --name cma-ssh --set install.operator=false --set images.operator.tag=0.1.12-local --set images.bootstrap.tag=0.1.12-local --set install.bootstrapIp=10.240.0.6 --set install.airgapProxyIp=10.240.0.7`

6. locally start operator `CMA_BOOTSTRAP_IP=10.240.0.6 CMA_NEXUS_PROXY_IP=10.240.0.7 ./cma-ssh`



## older - local debug

from repo root directory:

list available ops
`opctl ls`

run debug op
`opctl run debug`

it will ask for the ip of your machine.

create an `args.yml` file under ``.opspec` folder and place inputs in folder like

```
dockerSocket: /var/run/docker.sock
host-machine-ip: 192.168.1.129
```

after each debug run
`opctl run cleanup`

to remove the kind cluster
