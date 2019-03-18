set -o errexit

if [ $# = 0 ]; then
  echo >&2 "No packages specified"
  exit 1
fi

yum install -y $@
yum clean all -y
rm -rf \
   /var/cache/yum/* \
   /var/log/* \
   /tmp/* \
   /var/tmp/*