#!/bin/bash

set -e
set -o pipefail

build_root=/mnt/system
# shellcheck disable=SC2153
system_name="${SYSTEM_NAME:-"unnamed"}"
sqfs_image="$(echo "$PWD" | awk -F'/' '{fs = NF - 1; os = $fs; ver = NF; print os"-"$ver}')"
image_name=""

trap 'umount_sysdev' EXIT
sqfs["0"]="16.04:xenial"
sqfs["1"]="18.04:bionic"

umount_sysdev()
{
  umount "$build_root"/dev >/dev/null 2>&1 || true
  umount "$build_root"/sys >/dev/null 2>&1 || true
  umount "$build_root"/proc >/dev/null 2>&1 || true
}

get_image()
{
  local ver _ distro_name build_type match

  distro_name=""
  match=0

  if ! [[ "$sqfs_image" =~ ^(ubuntu|centos)-[0-9]+ ]]; then
    echo >&2 "Must run from distribution/version directory. You're in: $PWD"
    exit 1
  fi
  IFS='-' read -r distro ver <<< "$sqfs_image"

  for sq in "${sqfs[@]}"; do
    IFS=':' read -r v distro_name <<< "$sq"
    IFS='-' read -r ver build_type <<< "$ver"

    if [[ $v == "$ver" ]]; then
      match=1
      break
    fi
  done

  if [[ "$match" == 0 ]] || [[ "$distro_name" == "" ]]; then
    echo >&2 "cannot determine distribution name."
    exit 1
  fi

  image_name="$distro_name"-server-cloudimg-amd64.squashfs
  system_name="$distro-$ver-${build_type:-standard}".tar.gz

  mkdir -p iso
  wget -O iso/"$distro_name"-server-cloudimg-amd64.squashfs \
    https://cloud-images.ubuntu.com/"$distro_name"/current/"$image_name"
}

[[ -z "$build_root" ]] && exit 1
[[ "$build_root" == '/' ]] && exit 1
[[ $USER != "root" ]] && \
  {
    echo >&2 "Must be root"
    exit 1
  }

rm -rf "$build_root" > /dev/null 2>&1
mkdir -p "$build_root"/{tmp,etc}/
get_image

cp docker-install.sh "$build_root"/tmp/docker-install.sh
chown root "$build_root"/tmp/docker-install.sh && chgrp root "$build_root"/tmp/docker-install.sh
chmod +x "$build_root"/tmp/docker-install.sh

cp kubernetes-install.sh "$build_root"/tmp/kubernetes-install.sh
chown root "$build_root"/tmp/kubernetes-install.sh && chgrp root "$build_root"/tmp/kubernetes-install.sh
chmod +x "$build_root"/tmp/kubernetes-install.sh

unsquashfs -f -d "$build_root" iso/"$image_name"
mount -t proc proc "$build_root"/proc/
mount -t sysfs sys "$build_root"/sys/
mount -o bind /dev "$build_root"/dev

cp /etc/hosts "$build_root"/etc/hosts
# resolv.conf is a symlink to systemd runtime
mv "$build_root"/etc/resolv.conf "$build_root"/etc/resolv.conf.bak || true
echo 'nameserver 8.8.8.8' > "$build_root"/etc/resolv.conf

chroot "$build_root"/ /tmp/docker-install.sh
chroot "$build_root"/ /tmp/kubernetes-install.sh
# maas's preseed curtin_userdata cloud-init uses useradd
# to add users so set the default shell to /bin/bash instead
# the default of borne.
chroot "$build_root"/ /bin/sed -i.bak 's#SHELL=/bin/sh#SHELL=/bin/bash#' /etc/default/useradd

umount_sysdev

mv "$build_root"/etc/resolv.conf.bak "$build_root"/etc/resolv.conf || true
tar cpzf /var/tmp/"$system_name" -C "$build_root" .

rm -rf "$build_root"
