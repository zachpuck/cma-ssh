mv docker-install.sh /mnt/bionic/tmp/docker-install.sh
chown root /mnt/bionic/tmp/docker-install.sh && chgrp root /mnt/bionic/tmp/docker-install.sh
chmod +x /mnt/bionic/tmp/docker-install.sh
mv kubernetes-install.sh /mnt/bionic/tmp/kubernetes-install.sh
chown root /mnt/bionic/tmp/kubernetes-install.sh && chgrp root /mnt/bionic/tmp/kubernetes-install.sh
chmod +x /mnt/bionic/tmp/kubernetes-install.sh
unsquashfs -f -d /mnt/bionic bionic-server-cloudimg-amd64.squashfs
mount -t proc proc /mnt/bionic/proc/
mount -t sysfs sys /mnt/bionic/sys/
mount -o bind /dev /mnt/bionic/dev 

cp /etc/hosts /mnt/bionic/etc/hosts
mv /mnt/bionic/etc/resolv.conf /mnt/bionic/etc/resolv.conf.bak
echo 'nameserver 8.8.8.8' > /mnt/bionic/etc/resolv.conf

chroot /mnt/bionic/ /tmp/docker-install.sh
chroot /mnt/bionic/ /tmp/kubernetes-install.sh

mv /mnt/bionic/etc/resolv.conf.bak /mnt/bionic/etc/resolv.conf
umount /mnt/bionic/dev
umount /mnt/bionic/sys
umount /mnt/bionic/proc

tar cpzf bionic.tar.gz -C /mnt/bionic .
