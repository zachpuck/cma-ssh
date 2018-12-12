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
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	crypto "golang.org/x/crypto/ssh"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"text/template"
	"time"
)

type boostrapConfigInfo struct {
	ProxyIp       string
	BootstrapIp   string
	BootstrapPort string
}

type sshCommand func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error)

func RunSshCommand(kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	command sshCommand, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("RunSshCommand()")

	sshConfig := machineInstance.Spec.SshConfig
	secret := &corev1.Secret{}

	err := util.Retry(60, 10*time.Second, func() error {
		err := kubeClient.Get(
			context.Background(),
			client.ObjectKey{
				Namespace: machineInstance.GetNamespace(),
				Name:      sshConfig.Secret,
			},
			secret)
		if err != nil {
			log.Error(err,
				"could not find ssh key secret for machine "+machineInstance.GetName())
			return err
		}
		return nil
	})

	var host string
	if len(sshConfig.PublicHost) > 0 {
		host = sshConfig.PublicHost
	} else {
		host = sshConfig.Host
	}
	addr := fmt.Sprintf("%v:%v", host, sshConfig.Port)
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
			log.Error(err, "command "+cmd+" exited without status")
		case *crypto.ExitError:
			log.Error(err, "command "+cmd+" exited with failing status", "output", string(output[:]))
		}
	}

	return output, cmd, err
}

var IpAddr sshCommand = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	cr := &ssh.CommandRunner{}

	return cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "ip addr"},
	)
}

var InstallBootstrapRepo = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
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
	machineInstance *clusterv1alpha1.CnctMachine,
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

	// get the lowercased hostname for /etc/host
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " nginx -y"},
		ssh.Command{Cmd: "cat - > /etc/nginx/nginx.conf", Stdin: bytes.NewReader(configParsedNginx.Bytes())},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart nginx"},
		ssh.Command{Cmd: "systemctl enable nginx"},
		ssh.Command{Cmd: "systemctl stop firewalld"},
		ssh.Command{Cmd: "systemctl disable firewalld"},
		ssh.Command{Cmd: "echo -e '\n127.0.0.1   registry-1.docker.io gcr.io k8s.gcr.io quay.io\n' >> /etc/hosts"},
		ssh.Command{Cmd: "echo -e '" + templateData.ProxyIp + " sds.redii.net\n' >> /etc/hosts"},
		ssh.Command{Cmd: "echo -e '" + machineInstance.Spec.SshConfig.Host + " " +
			hostnameString + "\n' >> /etc/hosts"},
	)

	return nil, cmd, err
}

var InstallDocker = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
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
	machineInstance *clusterv1alpha1.CnctMachine,
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
	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "if [ $(getenforce) != 'Disabled' ]; then setenforce 0; fi"},
		ssh.Command{Cmd: "sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config"},
		ssh.Command{Cmd: "swapoff -a"},
		ssh.Command{Cmd: "if grep -Pq '^/dev/mapper/centos-swap' /etc/fstab; then " +
			"sed -ri.bak-$(date +%Y%m%dT%H%M%S) 's/(.*centos-swap.*)/#\\1/' /etc/fstab; fi"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " conntrack -y"},
		ssh.Command{Cmd: "modprobe br_netfilter"},
	)
	if err != nil {
		return nil, cmd, err
	}

	// read in k8s.conf
	k8sConf, err := asset.Assets.Open("/etc/sysctl.d/k8s.conf")
	if err != nil {
		log.Error(err, "error reading /etc/sysctl.d/k8s.conf")
		return nil, "", err
	}

	// run kubernetes install commands
	bootstrapRepoUrl := "http://" + templateData.BootstrapIp + ":" + templateData.BootstrapPort

	// for creates we use the version from machine status,
	// that is set as soon as machine starts being created
	k8sVersion := machineInstance.Status.KubernetesVersion
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/sysctl.d/k8s.conf", Stdin: k8sConf},
		ssh.Command{Cmd: "sysctl --system"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubelet-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubectl-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubeadm-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "sed -i 's/cgroup-driver=cgroupfs/cgroup-driver=systemd/g' " +
			"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"},
		ssh.Command{Cmd: "sed -i 's/cgroup-driver=systemd/cgroup-driver=systemd --runtime-cgroups=\\/systemd\\/system.slice " +
			"--kubelet-cgroups=\\/systemd\\/system.slice/g' /etc/systemd/system/kubelet.service.d/10-kubeadm.conf"},
		ssh.Command{Cmd: "mkdir -p /etc/kubernetes/"},
		ssh.Command{Cmd: "wget --output-document=/etc/kubernetes/kube-flannel.yml " + bootstrapRepoUrl + "/download/kube-flannel.yml"},
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl enable kubelet"},
		ssh.Command{Cmd: "systemctl restart kubelet"},
	)

	return nil, cmd, err
}

var KubeadmInit = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
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

	// get the lowercased hostname for /etc/host
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	err = util.Retry(60, 10*time.Second, func() error {
		for k, v := range machineInstance.Spec.Labels {
			cmd, err = cr.Run(
				client.Client,
				ssh.Command{Cmd: "kubectl --kubeconfig /etc/kubernetes/kubelet.conf label node " +
					hostnameString + " " + k + "=" + v},
			)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil, cmd, err
}

var KubeadmTokenCreate = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
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
	machineInstance *clusterv1alpha1.CnctMachine,
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
		ssh.Command{Cmd: "kubeadm join --token " +
			token +
			" --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification " +
			master},
	)

	// get the lowercased hostname for /etc/host
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	err = util.Retry(60, 10*time.Second, func() error {
		for k, v := range machineInstance.Spec.Labels {
			cmd, err = cr.Run(
				client.Client,
				ssh.Command{Cmd: "kubectl --kubeconfig /etc/kubernetes/kubelet.conf label node " +
					hostnameString + " " + k + "=" + v},
			)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil, cmd, err
}

var GetKubeConfig = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
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

var DeleteNode = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("delete node command")

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

	// check if kubelet is installed, uninstall
	log.Info("Checking wget for " + machineInstance.GetName())
	cmd, err := cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " wget"},
	)
	if err == nil {
		log.Info("Deleting wget for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " wget -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if kubeadm ins installed, uninstall
	log.Info("Checking kubeadm for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubeadm"},
	)
	if err == nil {
		log.Info("Deleting kubeadm for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "kubeadm reset --force"},
		)
		if err != nil {
			log.Info("kubeadm probably does not understand '--force' flag, trying without.")
			cmd, err = cr.Run(
				client.Client,
				ssh.Command{Cmd: "kubeadm reset"},
			)
			if err != nil {
				return nil, cmd, err
			}
		}

		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubeadm -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if kubelet is installed, uninstall
	log.Info("Checking kubelet for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubelet"},
	)
	if err == nil {
		log.Info("Deleting kubelet for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubelet -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if cni is installed, uninstall
	log.Info("Checking cni for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubernetes-cni"},
	)
	if err == nil {
		log.Info("Deleting cni for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubernetes-cni -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if kubectl is installed, uninstall
	log.Info("Checking kubectl for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubectl"},
	)
	if err == nil {
		log.Info("Deleting kubectl for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " kubectl -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if docker is installed, uninstall
	log.Info("Checking docker for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " docker"},
	)
	if err == nil {
		log.Info("Deleting docker for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " docker -y"},
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " docker-client -y"},
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " docker-common -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if nginx is installed, uninstall
	log.Info("Checking nginx for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " nginx"},
	)
	if err == nil {
		log.Info("Deleting nginx for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " nginx -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// check if conntrack is installed, uninstall
	log.Info("Checking conntrack for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum list installed --disablerepo='*' --enablerepo=" + bootstrapRepoName + " conntrack"},
	)
	if err == nil {
		log.Info("Deleting conntrack for " + machineInstance.GetName())
		cmd, err = cr.Run(
			client.Client,
			ssh.Command{Cmd: "yum remove --disablerepo='*' --enablerepo=" + bootstrapRepoName + " conntrack -y"},
		)
		if err != nil {
			return nil, cmd, err
		}
	}

	// delete bootstrap repo file
	log.Info("Deleting repo file for " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "rm -f /etc/yum.repos.d/" + bootstrapRepoName + ".repo"},
	)
	if err != nil {
		return nil, cmd, err
	}

	// delete folders
	log.Info("Deleting folders " + machineInstance.GetName())
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "rm -rf /etc/cni"},
		ssh.Command{Cmd: "rm -rf /etc/docker"},
		ssh.Command{Cmd: "rm -rf /etc/nginx"},
		ssh.Command{Cmd: "rm -rf /etc/sysconfig/docker"},
		ssh.Command{Cmd: "rm -rf /etc/ethertypes"},
		ssh.Command{Cmd: "rm -rf /etc/kubernetes"},
		ssh.Command{Cmd: "rm -rf /etc/systemd/system/kubelet.service.d"},
		ssh.Command{Cmd: "rm -rf /var/lib/cni"},
		ssh.Command{Cmd: "rm -rf /var/lib/docker"},
		ssh.Command{Cmd: "rm -rf /var/lib/dockershim"},
		ssh.Command{Cmd: "rm -rf /var/lib/etcd"},
		ssh.Command{Cmd: "rm -rf /var/lib/etcd2"},
		ssh.Command{Cmd: "rm -rf /var/lib/kubelet"},
	)
	if err != nil {
		return nil, cmd, err
	}

	return nil, cmd, nil
}

var UpgradeMaster = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Upgrade master command")

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
	// for updates we use the version from cluster, as that is what we are upgrading to
	k8sVersion := clusterInstance.Spec.KubernetesVersion

	// get the lowercased hostname
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	// get the bootstrap repo name
	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubeadm-" + k8sVersion + " -y --disableexcludes=kubernetes"},
		ssh.Command{Cmd: "kubeadm upgrade apply v" + k8sVersion + " -y"},
		ssh.Command{Cmd: "kubectl drain " + hostnameString + " --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubelet-" + k8sVersion + " -y --disableexcludes=kubernetes"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubectl-" + k8sVersion + " -y"},
	)
	if err != nil {
		return nil, cmd, err
	}

	// configure kubelet
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "echo -n 'KUBELET_EXTRA_ARGS=--cgroup-driver=systemd " +
			"--runtime-cgroups=/systemd/system.slice --kubelet-cgroups=/systemd/system.slice' > /etc/sysconfig/kubelet"},
	)
	if err != nil {
		return nil, cmd, err
	}

	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart kubelet"},
		ssh.Command{Cmd: "kubectl uncordon " + hostnameString + " --kubeconfig=/etc/kubernetes/admin.conf"},
	)

	return nil, cmd, err
}

var UpgradeNode = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.CnctMachine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Upgrade node command")

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
	// for updates we use the version from cluster, as that is what we are upgrading to
	k8sVersion := clusterInstance.Spec.KubernetesVersion

	// get the lowercased hostname
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	// get the bootstrap repo name
	bootstrapRepoName := templateData.BootstrapIp + "_" + templateData.BootstrapPort

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/kubernetes/admin.conf",
			Stdin: bytes.NewReader([]byte(commandArgs["admin.conf"]))},
		ssh.Command{Cmd: "kubectl drain " +
			hostnameString + " --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubeadm-" + k8sVersion + " -y --disableexcludes=kubernetes"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubelet-" + k8sVersion + " -y --disableexcludes=kubernetes"},
		ssh.Command{Cmd: "yum install --disablerepo='*' --enablerepo=" +
			bootstrapRepoName + " kubectl-" + k8sVersion + " -y"},
		ssh.Command{Cmd: "kubeadm upgrade node config --kubelet-version $(kubelet --version | cut -d ' ' -f 2)"},
	)
	if err != nil {
		return nil, cmd, err
	}

	// configure kubelet
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "echo -n 'KUBELET_EXTRA_ARGS=--cgroup-driver=systemd " +
			"--runtime-cgroups=/systemd/system.slice --kubelet-cgroups=/systemd/system.slice' > /etc/sysconfig/kubelet"},
	)
	if err != nil {
		return nil, cmd, err
	}

	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "systemctl daemon-reload"},
		ssh.Command{Cmd: "systemctl restart kubelet"},
		ssh.Command{Cmd: "kubectl uncordon " + hostnameString + " --kubeconfig=/etc/kubernetes/admin.conf"},
	)

	return nil, cmd, err
}

var DrainAndDeleteNode = func(client *ssh.Client, kubeClient client.Client,
	machineInstance *clusterv1alpha1.Machine,
	templateData boostrapConfigInfo, commandArgs map[string]string) ([]byte, string, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("Upgrade node command")

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

	// get the lowercased hostname
	hostname, cmd, err := cr.GetOutput(
		client.Client,
		ssh.Command{Cmd: "hostname"},
	)
	if err != nil {
		return nil, cmd, err
	}
	hostnameString := strings.ToLower(string(bytes.TrimSpace(hostname)[:]))

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	cmd, err = cr.Run(
		client.Client,
		ssh.Command{Cmd: "cat - > /etc/kubernetes/admin.conf",
			Stdin: bytes.NewReader([]byte(commandArgs["admin.conf"]))},
		ssh.Command{Cmd: "kubectl drain " +
			hostnameString + " --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"},
		ssh.Command{Cmd: "kubectl delete node " + hostnameString +
			" --kubeconfig=/etc/kubernetes/admin.conf"},
	)

	return nil, cmd, err
}