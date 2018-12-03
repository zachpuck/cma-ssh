### Build:

Build for local OSX dev:

```
make -f build/Makefile darwin
```

CRDs are generated in `./crd`  
RBAC is generated in `./rbac`

Helm chart under `./deployments/helm/cma-ssh` gets updated with the right CRDs and RBAC

### using bootstrap yum repo

Build docker containers

```
make -f build/Makefile docker-build
make -f build/Makefile docker-push
```

Install boostrap proxy into the fake command cluster:

```
helm install deployments/helm/cma-ssh --name cma-ssh --set images.bootstrap.tag=0.1.4-local --set install.operator=false
```

Yum repo will be served on `http://<NODE-IP>:30005`

### Andrew's local setup for testing ssh

create a centos7 virtualbox vm with host-only networking

on vm create a root password

    sudo passwd root

allow root login via ssh

    vi /etc/ssh/sshd_config

uncomment or add

    PermitRootLogin yes
    PasswordAuthentication yes

locally, send private key vm

    ssh-copy-id root@<virtualbox ip>

(optional) on vm remove password login

    PermitRootLogin without-password
    PasswordAuthentication no

locally, for proxy testing you can run squid proxy

    docker run --name squid -d -p 3128:3128 datadog/squid

change the machine in config/samples to have the virtualbox ip addr

create a secret called "cats" with the `private-key` field as the contents of the
private key for virtualbox vm

    make run

proxy is currently hardcoded to use my virtualbox subnet host ip
pkg/ssh/ssh.go:64
