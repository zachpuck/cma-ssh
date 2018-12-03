#!/bin/bash
chmod -R 0755 /yum
createrepo --update /yum
/usr/sbin/nginx -c /etc/nginx/nginx.conf -g "daemon off;"