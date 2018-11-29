package machine

import (
	"bufio"
	"bytes"
	"os"

	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh/asset"
)

func installNginx(client *ssh.Client) error {
	centos7Repo, err := asset.Assets.Open("/etc/yum.repos.d/centos7.repo")
	if err != nil {
		return err
	}
	nginxConf, err := asset.Assets.Open("/nginx.conf")
	if err != nil {
		return err
	}

	cr := ssh.CommandRunner{
		Stdout: bufio.NewWriter(os.Stdout),
		Stderr: bufio.NewWriter(os.Stderr),
	}

	return cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/yum.repos.d/centos7.repo", Stdin: centos7Repo},
		ssh.Command{Cmd: "yum install nginx -y"},
		ssh.Command{Cmd: "cat - > /etc/nginx/nginx.conf", Stdin: nginxConf},
		ssh.Command{Cmd: "systemctl restart nginx"},
	)
}

func installDocker(client *ssh.Client) error {
	var buf bytes.Buffer
	cr := ssh.CommandRunner{
		Stdout: &buf,
		Stderr: bufio.NewWriter(os.Stderr),
	}
	if err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "rpm -qa 'docker*'"},
	); err != nil {
		return err
	}

	if buf.Len() != 0 {
		return nil
	}

	daemonJSON, err := asset.Assets.Open("/etc/docker/daemon.json")
	if err != nil {
		return err
	}

	cr = ssh.CommandRunner{
		Stdout: bufio.NewWriter(os.Stdout),
		Stderr: bufio.NewWriter(os.Stderr),
	}

	return cr.Run(
		client.Client,
		ssh.Command{Cmd: "mkdir -p /etc/docker"},
		ssh.Command{Cmd: "cat - > /etc/docker/daemon.json", Stdin: daemonJSON},
		ssh.Command{Cmd: "mkdir -p /etc/systemd/system/docker.service.d"},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart docker"},
	)
}
