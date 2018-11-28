# opctl

requires [opctl](https://opctl.io/docs/getting-started/opctl.html) installed

# usage

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
