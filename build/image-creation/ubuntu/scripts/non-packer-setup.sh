#!/bin/bash

set -e
trap 'umount_sysdev' EXIT

umount_sysdev()
{
  umount /mnt/bionic/dev || true
  umount /mnt/bionic/sys || true
  umount /mnt/bionic/proc || true
}

mkdir -p /mnt/bionic/{tmp,etc}/

cp ../scripts/docker-install.sh /mnt/bionic/tmp/docker-install.sh
chown root /mnt/bionic/tmp/docker-install.sh && chgrp root /mnt/bionic/tmp/docker-install.sh
chmod +x /mnt/bionic/tmp/docker-install.sh

cp ../scripts/kubernetes-install.sh /mnt/bionic/tmp/kubernetes-install.sh
chown root /mnt/bionic/tmp/kubernetes-install.sh && chgrp root /mnt/bionic/tmp/kubernetes-install.sh
chmod +x /mnt/bionic/tmp/kubernetes-install.sh

unsquashfs -f -d /mnt/bionic iso/bionic-server-cloudimg-amd64.squashfs
mount -t proc proc /mnt/bionic/proc/
mount -t sysfs sys /mnt/bionic/sys/
mount -o bind /dev /mnt/bionic/dev 

cp /etc/hosts /mnt/bionic/etc/hosts
# resolv.conf is a symlink to systemd runtime
mv /mnt/bionic/etc/resolv.conf /mnt/bionic/etc/resolv.conf.bak || true
echo 'nameserver 8.8.8.8' > /mnt/bionic/etc/resolv.conf

chroot /mnt/bionic/ /tmp/docker-install.sh
chroot /mnt/bionic/ /tmp/kubernetes-install.sh

umount_sysdev

mv /mnt/bionic/etc/resolv.conf.bak /mnt/bionic/etc/resolv.conf || true
tar cpzf /var/tmp/bionic.tar.gz -C /mnt/bionic .
