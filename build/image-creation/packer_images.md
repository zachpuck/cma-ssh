# Building images with packer

We are building images for MaaS; a general service which can provision operating systems to bare-metal as a service. Thus, we may have a need to many images generally. This article will explain the process we use to build custom images for MaaS using packer. The use of Packer seemed logical for purpose now, but we did notice that using Packer severely increases the time to 'baking' an image.

The following documentation assumes you're starting from nothing. This repo already has the files for Ubuntu 16.04 and 18.04. However, if you need to extend to a different distribution or version of an exhisting distribution, this documentation will attempt to walk you through that process. If you don't need to create anything new then please skip down to [#Run_packer](Run packer).

## Steps to build

### Download Packer.

You can download packer from [here](https://packer.io/downloads.html).

### Choose a builder.

I prefer to do the image build on my computer so I use the Vagrant builder.
You can also use AWS EC2, Azure, Google Cloud etc to build images.

### Create a json template file for the build.

```js
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

## Provisioning Scripts Directory Layout

This directory is created to create the versioned distribution dictated by the directory under which the packer assets reside. For instance, the path `<repo_root>/build/ubuntu/18.04` will create the image used for the stated distro and version.

### Download a base image to use.

It was deemed a reasonably good idea to use the online MaaS image repo as a starting point for a vanilla OS image. The [Ubuntu Server Cloud Image](https://cloud-images.ubuntu.com/bionic/current/) for Bionic will be used for the purpose of illustration. Download a squashfs image to your local directory.

TODO: should image download be automatic?

### Create script to unsqaush the fs

[script](<repo_root>/build/image-creation/ubuntu/scripts/setup.sh) that handles unsquashing and squashing.

### Create script to install components

[docker](<repo_root>/build/image-creation/ubuntu/scripts/docker-install.sh) and [kubernetes](<repo_root>/build/image-creation/ubuntu/scripts/kubernetes-install.sh) install scripts

### Add provisioners to copy files to builder.

```js
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

```js
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

```js
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

### Run packer

Make sure you in the propre directory for the distribution you want to build and then run: `packer build golden.json`

wait many hours.

Note that this method will not work if you're attempting to run vagrant from with a virtual machine using Virtualbox. Virtualbox does not expose the parent machine's processor exentions for virtual processing.

### Image file should be output to your local directory.

# Importing into MaaS

## Pre-requisites

  * Must have the maas-cli installed:
    * Ubuntu: `$ sudo apt install maas-cli`
    * *Mac: Not available
  * Have [properly configured administrator access to MaaS](https://docs.maas.io/2.5/en/installconfig-webui) via CLI.

  If you're unable to install maas-cli on your platform, you can use maas-cli on the maas master `192.168.2.24`.

### Import New Image from MaaS Master *(only method available if you can't use local maas-cli)*

  The image will need to get moved to the MaaS master server for import. Gaining access to the MaaS master is beyond the scope of this document. One can view [Smith Tower VPN Access](https://samsung-cnct.atlassian.net/wiki/spaces/AG/pages/156467219/Smith+Tower+VPN+Access) for that information.

  You will need to `scp` the image to the master. An example of how to do that is:
  ```bash
  $ scp -i </path/to/identity key> </path/to/fresh_image.tgz> ubuntu@192.168.2.24:/var/tmp
  ```

  Once the image is on the MaaS master, you will need to ssh to the master:
  ```bash
  $ ssh -i /path/to/<identity key> ubuntu@192.168.2.24
  ```

  Now follow the import instructions in [](#Import_New_Image_to_MaaS_from_your_workstation)

### Import New Image to MaaS from your workstation

  The following steps are [documented here](https://samsung-cnct.atlassian.net/wiki/spaces/MAAS/pages/323256342/Import+Custom+Image+to+MaaS).

  The following instruction reference a `profile`. This is generally your administrative **username**. It is specifically the name you or someone else provided when creating your maas priv access. For the purposes of this document, the word `profile` will be used for demonstration purposes.

  The image that was created should be a gzipped tarball. If it isn't then your mileage may vary on the success of this command. The following command will import the image that was just created:

  ```bash
  $ maas <profile> boot-resources create name=custom/<valid image name> title="<valid title>" architecture=amd64/generic content@=root.tgz

  ```

  One can verify the image is available by checking the MaaS GUI http://192.168.2.24:5240/MAAS/#/images and clicking the "Custom" radio button.

### Deploy the new image to a machine in the MaaS Node Pool

  The following command can then be used to deploy the image to be installed on a single machine. The `user_data=.*` is optional. This command is given as an example only.

  ```bash
  $ maas <profile> machine deploy <machine id> distro_series=ubuntu-18.10-tgz-import-test user_data=$(base64 ts.sh | tr -d ' ')
  ```
