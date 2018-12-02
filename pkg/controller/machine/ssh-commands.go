package machine

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh/asset"
	crypto "golang.org/x/crypto/ssh"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"text/template"
)

type boostrapConfigInfo struct {
	ProxyIp string
}

type sshCommand func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error)

func RunSshCommand(kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	command sshCommand, commandArgs map[string]string) (string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("RunSshCommand()")

	sshConfig := machineInstance.Spec.SshConfig
	secret := &corev1.Secret{}
	err := kubeClient.Get(
		context.Background(),
		client.ObjectKey{
			Namespace: machineInstance.GetNamespace(),
			Name:      sshConfig.Secret,
		},
		secret)
	if err != nil {
		log.Error(err,
			"could not find object secret", "secret", sshConfig.Secret)
		return "", err
	}

	addr := fmt.Sprintf("%v:%v", sshConfig.Host, sshConfig.Port)
	sshClient, err := ssh.NewClient(addr, sshConfig.Username, secret.Data["private-key"])
	if err != nil {
		return "", err
	}

	proxyIp, present := os.LookupEnv("CMA_NEXUS_PROXY_IP")
	if !present {
		// TODO: this is not great...
		proxyIp = "182.195.81.113"
	}

	templateInfo := boostrapConfigInfo{
		ProxyIp: proxyIp,
	}

	output, err := command(sshClient, kubeClient, machineInstance, templateInfo, commandArgs)
	if err != nil {
		switch err.(type) {
		case *crypto.ExitMissingError:
			log.Error(err, "command exited without status", "command")
		case *crypto.ExitError:
			log.Error(err, "command exited with failing status")
		}
	}

	if output == nil {
		return "", err
	} else {
		return string(output[:]), err
	}
}

var IpAddr sshCommand = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	cr := &ssh.CommandRunner{}

	return cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "ip addr"},
	)
}

var InstallNginx = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Install nginx command")

	centosConf, err := asset.Assets.Open("/etc/yum.repos.d/centos7.repo")
	if err != nil {
		return nil, err
	}

	nginxConf, err := asset.Assets.Open("/etc/nginx/nginx.conf")
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(nginxConf)
	if err != nil {
		return nil, err
	}
	configTemplateNginx, err := template.New("nginx-config").Parse(string(buf[:]))
	if err != nil {
		return nil, err
	}
	var configParsedNginx bytes.Buffer
	if err := configTemplateNginx.Execute(&configParsedNginx, templateData); err != nil {
		return nil, err
	}

	buf, err = ioutil.ReadAll(centosConf)
	if err != nil {
		return nil, err
	}
	configTemplateCentos, err := template.New("centos-config").Parse(string(buf[:]))
	if err != nil {
		return nil, err
	}
	var configParsedCentos bytes.Buffer
	if err := configTemplateCentos.Execute(&configParsedCentos, templateData); err != nil {
		return nil, err
	}

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)
	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/yum.repos.d/centos7.repo", Stdin: bytes.NewReader(configParsedCentos.Bytes())},
		ssh.Command{Cmd: "yum install nginx -y"},
		ssh.Command{Cmd: "cat - > /etc/nginx/nginx.conf", Stdin: bytes.NewReader(configParsedNginx.Bytes())},
		ssh.Command{Cmd: "systemctl restart nginx"},
		ssh.Command{Cmd: "systemctl enable nginx"},
	)

	return nil, err
}

var InstallDocker = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Install docker command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	isDockerInstalled, err := cr.GetOutput(client.Client, ssh.Command{Cmd: "rpm -qa 'docker*'"})
	if err != nil {
		return nil, err
	}

	if len(isDockerInstalled) > 0 {
		return nil, nil
	}

	dockerConf, err := asset.Assets.Open("/etc/docker/daemon.json")
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(dockerConf)
	if err != nil {
		return nil, err
	}
	configTemplateDocker, err := template.New("docker-config").Parse(string(buf[:]))
	if err != nil {
		return nil, err
	}
	var configParsedDocker bytes.Buffer
	if err := configTemplateDocker.Execute(&configParsedDocker, templateData); err != nil {
		return nil, err
	}

	return nil, cr.Run(
		client.Client,
		ssh.Command{Cmd: "mkdir -p /etc/docker"},
		ssh.Command{Cmd: "cat - > /etc/docker/daemon.json", Stdin: bytes.NewReader(configParsedDocker.Bytes())},
		ssh.Command{Cmd: "mkdir -p /etc/systemd/system/docker.service.d"},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart docker"},
		ssh.Command{Cmd: "systemctl enable docker"},
	)
}

var InstallKubernetes = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Install kubernetes command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	// selinux disable
	err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "setenforce 0"},
		ssh.Command{Cmd: "sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config"},
		ssh.Command{Cmd: "swapoff -a"},
	)
	if err != nil {
		log.Error(err, "error running selinux disable")
		return nil, err
	}

	bootstrapImage, present := os.LookupEnv("CMA_BOOTSTRAP_IMAGE")
	if !present {
		// TODO: this is not great...
		bootstrapImage = "quay.io/samsung_cnct/cma-ssh-bootstrap:latest"
	}

	// pull bootstrap image and get a docker image id to copy from
	bootstrapImageId, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "docker create " + bootstrapImage})
	if err != nil {
		log.Error(err, "error pulling boostrap image")
		return nil, err
	}

	// get the kubernetes version to use
	clusterInstance, err := getCluster(kubeClient, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		log.Error(err, "error getting cluster instance")
		return nil, err
	}

	// read in kubernetes local repo source
	k8sRepo, err := asset.Assets.Open("/etc/yum.repos.d/kubernetes-local.repo")
	if err != nil {
		log.Error(err, "error reading kubernetes yum repo config")
		return nil, err
	}

	// read in k8s.conf
	k8sConf, err := asset.Assets.Open("/etc/sysctl.d/k8s.conf")
	if err != nil {
		log.Error(err, "error reading /etc/sysctl.d/k8s.conf")
		return nil, err
	}

	// run docker copy commands and kubernetes install commands
	err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "docker cp " + string(bootstrapImageId[:]) +
			":/resources/rpms/" + clusterInstance.Spec.KubernetesVersion + " /etc/kubernetes/rpms"},
		ssh.Command{Cmd: "docker cp " + string(bootstrapImageId[:]) +
			":/resources/yaml/kube-flannel.yml /etc/kubernetes/kube-flannel.yml"},
		ssh.Command{Cmd: "createrepo /etc/kubernetes/rpms"},
		ssh.Command{Cmd: "cat - > /etc/yum.repos.d/kubernetes-local.repo", Stdin: k8sRepo},
		ssh.Command{Cmd: "yum --disablerepo='*' --enablerepo=kubernetes-local -y install kubelet"},
		ssh.Command{Cmd: "yum --disablerepo='*' --enablerepo=kubernetes-local -y install kubectl"},
		ssh.Command{Cmd: "yum --disablerepo='*' --enablerepo=kubernetes-local -y install kubeadm"},
		ssh.Command{Cmd: "cat - > /etc/sysctl.d/k8s.conf", Stdin: k8sConf},
		ssh.Command{Cmd: "sysctl --system"},
		ssh.Command{Cmd: "systemctl enable kubelet"},
		ssh.Command{Cmd: "systemctl start kubelet"},
	)
	if err != nil {
		log.Error(err, "error running docker copy and kubernetes install")
		return nil, err
	}

	return nil, nil
}

var KubeadmInit = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Kubeadm init command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	// get the kubernetes version to use
	clusterInstance, err := getCluster(kubeClient, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		log.Error(err, "error getting cluster instance")
		return nil, err
	}

	// kubeadm init
	err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "kubeadm init --pod-network-cidr=" +
			clusterInstance.Spec.ClusterNetwork.Pods.CIDRBlock +
			"--kubernetes-version=" + clusterInstance.Spec.KubernetesVersion},
		ssh.Command{Cmd: "kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f /etc/kubernetes/kube-flannel.yml --force=true"},
	)
	if err != nil {
		log.Error(err, "error kubeadm init")
		return nil, err
	}

	return nil, nil
}

var CheckKubeadm = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Check kubeadm command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	// kubeadm check
	err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "which kubeadm"},
	)
	if err != nil {
		log.Error(err, "error kubeadm check")
		return nil, err
	}

	return nil, nil
}

var KubeadmTokenCreate = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Kubeadm token create command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	// kubeadm check
	token, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "kubeadm token create --description " + machineInstance.GetName()},
	)
	if err != nil {
		log.Error(err, "error kubeadm token create")
		return nil, err
	}

	return token, nil
}

var KubeadmJoin = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Kubeadm join command")

	bout := bufio.NewWriter(os.Stdout)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stdout writer")
		}
	}(bout)

	berr := bufio.NewWriter(os.Stderr)
	defer func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			log.Error(err, "could not flush os.Stderr writer")
		}
	}(berr)

	token := commandArgs["token"]
	master := commandArgs["master"] + ":" + common.ApiEnpointPort

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	// kubeadm join
	err := cr.Run(
		client.Client,
		ssh.Command{Cmd: " kubeadm join --token " +
			token +
			" --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification " +
			master},
	)
	if err != nil {
		log.Error(err, "error kubeadm join")
		return nil, err
	}

	return nil, nil
}
