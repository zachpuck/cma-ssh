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
	ProxyIp       string
	BootstrapIp   string
	BootstrapPort string
}

type sshCommand func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error)

func RunSshCommand(kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	command sshCommand, commandArgs map[string]string) ([]byte, string, error) {
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
		return nil, "", err
	}

	addr := fmt.Sprintf("%v:%v", sshConfig.Host, sshConfig.Port)
	sshClient, err := ssh.NewClient(addr, sshConfig.Username, secret.Data["private-key"])
	if err != nil {
		return nil, "", err
	}

	proxyIp, present := os.LookupEnv("CMA_NEXUS_PROXY_IP")
	if !present {
		// TODO: this is not great...
		proxyIp = "182.195.81.113"
	}

	bootstrapIp, present := os.LookupEnv("CMA_BOOTSTRAP_IP")
	if !present {
		// TODO: this is not great...
		bootstrapIp = "192.168.64.24"
	}

	bootstrapPort, present := os.LookupEnv("CMA_BOOTSTRAP_PORT")
	if !present {
		// TODO: this is not great...
		bootstrapPort = "30005"
	}

	templateInfo := boostrapConfigInfo{
		ProxyIp:       proxyIp,
		BootstrapIp:   bootstrapIp,
		BootstrapPort: bootstrapPort,
	}

	output, cmd, err := command(sshClient, kubeClient, machineInstance, templateInfo, commandArgs)
	if err != nil {
		switch err.(type) {
		case *crypto.ExitMissingError:
			log.Error(err, "command exited without status")
		case *crypto.ExitError:
			log.Error(err, "command exited with failing status", "output", string(output[:]))
		}
	}

	return output, cmd, err
}

var IpAddr sshCommand = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	cr := &ssh.CommandRunner{}

	return cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "ip addr"},
	)
}

var InstallBootstrapRepo = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Install bootstrap repo command")

	bootstrapConf, err := asset.Assets.Open("/etc/yum.repos.d/bootstrap.repo")
	if err != nil {
		return nil, "", err
	}

	buf, err := ioutil.ReadAll(bootstrapConf)
	if err != nil {
		return nil, "", err
	}
	configTemplateBootstrap, err := template.New("bootstrap-config").Parse(string(buf[:]))
	if err != nil {
		return nil, "", err
	}
	var configParsedBootstrap bytes.Buffer
	if err := configTemplateBootstrap.Execute(&configParsedBootstrap, templateData); err != nil {
		return nil, "", err
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

	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/yum.repos.d/" + bootstrapRepoName + ".repo",
			Stdin: bytes.NewReader(configParsedBootstrap.Bytes())},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " wget -y"},
	)

	return nil, cmd, err
}

var InstallNginx = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Install nginx command")

	nginxConf, err := asset.Assets.Open("/etc/nginx/nginx.conf")
	if err != nil {
		return nil, "", err
	}

	buf, err := ioutil.ReadAll(nginxConf)
	if err != nil {
		return nil, "", err
	}
	configTemplateNginx, err := template.New("nginx-config").Parse(string(buf[:]))
	if err != nil {
		return nil, "", err
	}
	var configParsedNginx bytes.Buffer
	if err := configTemplateNginx.Execute(&configParsedNginx, templateData); err != nil {
		return nil, "", err
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

	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " nginx -y"},
		ssh.Command{Cmd: "cat - > /etc/nginx/nginx.conf", Stdin: bytes.NewReader(configParsedNginx.Bytes())},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart nginx"},
		ssh.Command{Cmd: "systemctl enable nginx"},
		ssh.Command{Cmd: "echo -e '\n127.0.0.1   registry-1.docker.io gcr.io k8s.gcr.io quay.io\n' >> /etc/hosts"},
	)

	return nil, cmd, err
}

var InstallDocker = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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

	dockerConf, err := asset.Assets.Open("/etc/docker/daemon.json")
	if err != nil {
		return nil, "", err
	}

	buf, err := ioutil.ReadAll(dockerConf)
	if err != nil {
		return nil, "", err
	}
	configTemplateDocker, err := template.New("docker-config").Parse(string(buf[:]))
	if err != nil {
		return nil, "", err
	}
	var configParsedDocker bytes.Buffer
	if err := configTemplateDocker.Execute(&configParsedDocker, templateData); err != nil {
		return nil, "", err
	}

	cr := ssh.CommandRunner{
		Stdout: bout,
		Stderr: berr,
	}

	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " audit -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " device-mapper-persistent-data -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " lvm2 -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " docker -y"},
		ssh.Command{Cmd: "sed -i 's/native.cgroupdriver=cgroupfs/native.cgroupdriver=systemd/g' /usr/lib/systemd/system/docker.service"},
		ssh.Command{Cmd: "mkdir -p /etc/docker"},
		ssh.Command{Cmd: "cat - > /etc/docker/daemon.json", Stdin: bytes.NewReader(configParsedDocker.Bytes())},
		ssh.Command{Cmd: "mkdir -p /etc/systemd/system/docker.service.d"},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart docker"},
		ssh.Command{Cmd: "systemctl enable docker"},
	)

	return nil, cmd, err
}

var InstallKubernetes = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "if [ $(getenforce) != 'Disabled' ]; then setenforce 0; fi"},
		ssh.Command{Cmd: "sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config"},
		ssh.Command{Cmd: "swapoff -a"},
	)
	if err != nil {
		return nil, cmd, err
	}

	// get the kubernetes version to use
	clusterInstance, err := getCluster(kubeClient, machineInstance.GetNamespace(), machineInstance.Spec.ClusterRef)
	if err != nil {
		log.Error(err, "error getting cluster instance")
		return nil, "", err
	}

	// read in k8s.conf
	k8sConf, err := asset.Assets.Open("/etc/sysctl.d/k8s.conf")
	if err != nil {
		log.Error(err, "error reading /etc/sysctl.d/k8s.conf")
		return nil, "", err
	}

	// run kubernetes install commands
	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	bootstrapRepoUrl := "http://" + templateData.BootstrapIp + ":" + templateData.BootstrapPort
	k8sVersion := clusterInstance.Spec.KubernetesVersion
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/sysctl.d/k8s.conf", Stdin: k8sConf},
		ssh.Command{Cmd: "sysctl --system"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubelet-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubectl-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubeadm-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "sed -i 's/cgroup-driver=cgroupfs/cgroup-driver=systemd/g' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf"},
		ssh.Command{Cmd: "mkdir -p /etc/kubernetes/"},
		ssh.Command{Cmd: "wget --output-document=/etc/kubernetes/kube-flannel.yml " + bootstrapRepoUrl + "/download/kube-flannel.yml"},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl enable kubelet"},
		ssh.Command{Cmd: "systemctl restart kubelet"},
	)

	return nil, cmd, err
}

var KubeadmInit = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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
		return nil, "", err
	}

	// kubeadm init
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "kubeadm init --pod-network-cidr=" +
			clusterInstance.Spec.ClusterNetwork.Pods.CIDRBlock +
			" --kubernetes-version=" + clusterInstance.Spec.KubernetesVersion},
		ssh.Command{Cmd: "kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f /etc/kubernetes/kube-flannel.yml --force=true"},
	)

	return nil, cmd, err
}

var CheckKubeadm = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "which kubeadm"},
	)

	return nil, cmd, err
}

var KubeadmTokenCreate = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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
	return cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "kubeadm token create --description " + machineInstance.GetName()},
	)
}

var KubeadmJoin = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
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
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: " kubeadm join --token " +
			token +
			" --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification " +
			master},
	)

	return nil, cmd, err
}

var GetKubeConfig = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Get kubeconfig")

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
	return cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "cat /etc/kubernetes/admin.conf"},
	)
}
