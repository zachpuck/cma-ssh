package machine

import (
	"bufio"
	"os"
	"strings"

	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
)

var (
	centos7Repo = strings.NewReader(`[epel]
name=Extra Packages for Enterprise Linux 7 - $basearch
baseurl=http://182.195.81.113:9468/repository/cmp-yum-epel/$releasever/$basearch
failovermethod=priority
enabled=1
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-7

[epel-debuginfo]
name=Extra Packages for Enterprise Linux 7 - $basearch - Debug
baseurl=http://182.195.81.113:9468/repository/cmp-yum-epel/$releasever/$basearch/debug
failovermethod=priority
enabled=0
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-7

[epel-source]
name=Extra Packages for Enterprise Linux 7 - $basearch - Source
baseurl=http://182.195.81.113:9468/repository/cmp-yum-epel/$releasever/SRPMS
failovermethod=priority
enabled=0
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-7

[base]
name=Nexus Repository
baseurl=http://182.195.81.113:9468/repository/cmp-yum-centos/$releasever/os/$basearch/
enabled=1
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7
priority=1

#released updates
[updates]
name=CentOS-$releasever - Updates
baseurl=http://182.195.81.113:9468/repository/cmp-yum-centos/$releasever/updates/$basearch/
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7

#additional packages that may be useful
[extras]
name=CentOS-$releasever - Extras
baseurl=http://182.195.81.113:9468/repository/cmp-yum-centos/$releasever/extras/$basearch/
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7

#additional packages that extend functionality of existing packages
[centosplus]
name=CentOS-$releasever - Plus
baseurl=http://182.195.81.113:9468/repository/cmp-yum-centos/$releasever/centosplus/$basearch/
enabled=0
gpgcheck=0
#gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7
`)
	nginxConf = strings.NewReader(`user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
  worker_connections  1024;
}

http {
  server {
    server_name registry-1.docker.io;
    listen 80;
    location / {
      proxy_pass http://182.195.81.113:9401;
      proxy_redirect off;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      client_max_body_size 10G;
      client_body_buffer_size 128k;
      proxy_connect_timeout 90;
      proxy_send_timeout 90;
      proxy_read_timeout 90;
      proxy_buffer_size 4k;
      proxy_buffers 4 32k;
      proxy_busy_buffers_size 64k;


      proxy_temp_file_write_size 64k;
    }
  }
  server {
    server_name gcr.io;
    listen 80;
    location / {
      proxy_pass http://182.195.81.113:9402;
      proxy_redirect off;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      client_max_body_size 10G;
      client_body_buffer_size 128k;
      proxy_connect_timeout 90;
      proxy_send_timeout 90;
      proxy_read_timeout 90;
      proxy_buffer_size 4k;
      proxy_buffers 4 32k;
      proxy_busy_buffers_size 64k;
      proxy_temp_file_write_size 64k;
    }
  }
  server {
    server_name k8s.gcr.io;
    listen 80;
    location / {
      proxy_pass http://182.195.81.113:9403;
      proxy_redirect off;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      client_max_body_size 10G;
      client_body_buffer_size 128k;
      proxy_connect_timeout 90;
      proxy_send_timeout 90;
      proxy_read_timeout 90;
      proxy_buffer_size 4k;
      proxy_buffers 4 32k;
      proxy_busy_buffers_size 64k;
      proxy_temp_file_write_size 64k;
    }
  }
  server {
    server_name quay.io;
    listen 80;
    location / {
      proxy_pass http://182.195.81.113:9404;
      proxy_redirect off;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      client_max_body_size 10G;
      client_body_buffer_size 128k;
      proxy_connect_timeout 90;
      proxy_send_timeout 90;
      proxy_read_timeout 90;
      proxy_buffer_size 4k;
      proxy_buffers 4 32k;
      proxy_busy_buffers_size 64k;
      proxy_temp_file_write_size 64k;
    }
  }
}
`)
)

func installNginx(client *ssh.Client) error {
	cr := ssh.CommandRunner{
		Stdout: bufio.NewWriter(os.Stdout),
		Stderr: bufio.NewWriter(os.Stderr),
	}

	return cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/yum.repos.d/centos7.repo", Stdin: centos7Repo},
		ssh.Command{Cmd: "yum install nginx -y"},
		ssh.Command{Cmd: "cat - > /etc/nginx/nginx.conf", Stdin: nginxConf},
		ssh.Command{Cmd: "service nginx restart"},
	)
}
