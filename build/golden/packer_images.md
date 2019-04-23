# Building images with packer

This article will explain the process we use to build custom images for MaaS
using packer.

## Steps to build

### Download Packer.

You can download packer from [here](https://packer.io/downloads.html).

### Choose a builder.

I prefer to do the image build on my computer so I use the Vagrant builder.
You can also use AWS EC2, Azure, Google Cloud etc to build images.

### Create a json template file for the build.

``` js
{
  "builders": [
    // one or many builders here
  ],
  "provisioners": [
    // your scripts to create the image
    // ...
    // download your resulting image
    {
      "type": "file",
      "source": "bionic.squashfs",
      "destination": "bionic.squashfs",
      "direction": "download"
    }
  ],
  "post-processors": [
    [
      {
        "type": "artifice",
        "files": ["bionic.squashfs"]
      }
    ]
  ]
}
```

### Run packer

command is `packer build golden.json`
wait many hours

### Image file should be output to your local directory.
    
## Our provisioning scripts

### Download a base image to use.

I chose [Ubuntu Server Cloud Image](https://cloud-images.ubuntu.com/bionic/current/). Download a squashfs image to your local
directory.

TODO: should image download be automatic?

### Create script to unsqaush the fs

[script](./setup.sh) that handles unsquashing and squashing.

### Create script to install components

[docker](./docker-install.sh) and [kubernetes](./kubernetes-install.sh) install scripts

### Add provisioners to copy files to builder.
    
``` js
[
  // ...
    {
      "type": "file",
      "source": "iso/bionic-server-cloudimg-amd64.squashfs",
      "destination": "bionic-server-cloudimg-amd64.squashfs"
    },
    {
      "type": "file",
      "source": "docker-install.sh",
      "destination": "docker-install.sh"
    },
    {
      "type": "file",
      "source": "kubernetes-install.sh",
      "destination": "kubernetes-install.sh"
    },
  // ...
]
```

### Add provisioner to run main script.
    
``` js
[
  // ...
    {
      "type": "shell",
      "execute_command": "echo 'vagrant' | {{.Vars}} sudo -S -E bash '{{.Path}}'",
      "script": "setup.sh"
    },
  // ...
]
```

### Add provisioner to download squashfs.

The last provisioner in should download the squashfs to the local dir.
    
``` js
[
  // ...
    {
      "type": "file",
      "source": "bionic.squashfs",
      "destination": "bionic.squashfs",
      "direction": "download"
    }
]
```
