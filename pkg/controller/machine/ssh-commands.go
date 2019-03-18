package machine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
	clusterv1alpha1 "github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/v1alpha1"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh"
	"github.com/samsung-cnct/cma-ssh/pkg/ssh/asset"
	"github.com/samsung-cnct/cma-ssh/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bootstrapConfig struct {
	ProxyIp       string
	BootstrapIp   string
	BootstrapPort string
}

type CmdConfig struct {
	sshClient       *ssh.Client
	clusterInstance *clusterv1alpha1.CnctCluster
	machineInstance *clusterv1alpha1.CnctMachine
	templateData    bootstrapConfig
}

func NewCmdConfig(kubeClient client.Client, machineInstance *clusterv1alpha1.CnctMachine, privateKey []byte) (CmdConfig, error) {
	sshConfig := machineInstance.Spec.SshConfig

	var host string
	if len(sshConfig.PublicHost) > 0 {
		host = sshConfig.PublicHost
	} else {
		host = sshConfig.Host
	}
	addr := fmt.Sprintf("%v:%v", host, sshConfig.Port)
	sshClient, err := ssh.NewClient(addr, sshConfig.Username, privateKey)
	if err != nil {
		return CmdConfig{}, err
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

	templateInfo := bootstrapConfig{
		ProxyIp:       proxyIp,
		BootstrapIp:   bootstrapIp,
		BootstrapPort: bootstrapPort,
	}

	clusterKey := client.ObjectKey{
		Namespace: machineInstance.GetNamespace(),
		Name:      machineInstance.Spec.ClusterRef,
	}

	clusterInstance := &clusterv1alpha1.CnctCluster{}
	if err := kubeClient.Get(context.Background(), clusterKey, clusterInstance); err != nil {
		return CmdConfig{}, err
	}

	return CmdConfig{
		sshClient:       sshClient,
		machineInstance: machineInstance,
		clusterInstance: clusterInstance,
		templateData:    templateInfo,
	}, nil

}

func parseAndExecute(file string, data bootstrapConfig) (*bytes.Buffer, error) {
	f, err := asset.Assets.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tmplBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(file).Parse(string(tmplBytes))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return &buf, nil
}

func InstallBootstrapRepo(cfg CmdConfig, args map[string]string) error {
	const file = "/etc/yum.repos.d/sds_bootstrap.repo"
	buf, err := parseAndExecute(file, cfg.templateData)
	if err != nil {
		return err
	}

	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)
	br.Run(
		writeToFile(file, buf),
		yum("wget"),
	)

	return br.Err()
}

func InstallNginx(cfg CmdConfig, args map[string]string) error {
	const file = "/etc/nginx/nginx.conf"
	nginx, err := parseAndExecute("/etc/nginx/nginx.conf", cfg.templateData)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	cmd := ssh.Cmd{
		Command: "hostname",
		Stdout:  &buf,
	}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}

	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	const newHostFmt = `
127.0.0.1 registry-1.docker.io gcr.io k8s.gcr.io quay.io
%s sds.redii.net
%s %s
`

	newHosts := strings.NewReader(
		fmt.Sprintf(
			newHostFmt,
			cfg.templateData.ProxyIp,
			cfg.machineInstance.Spec.SshConfig.Host,
			string(hostname),
		),
	)

	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)

	br.Run(
		yum("nginx"),
		writeToFile(file, nginx),
		ssh.Command("systemctl daemon-reload"),
		ssh.Command("systemctl restart nginx"),
		ssh.Command("systemctl enable nginx"),
		ssh.Command("systemctl stop firewalld"),
		ssh.Command("systemctl disable firewalld"),
		ssh.CommandWithInput("cat - >> /etc/hosts", newHosts),
	)

	return br.Err()
}

func InstallDocker(cfg CmdConfig, args map[string]string) error {
	const file = "/etc/docker/daemon.json"
	daemonJson, err := parseAndExecute(file, cfg.templateData)
	if err != nil {
		return err
	}

	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)
	br.Run(
		yum("docker-ce-18.06.2.ce"),
		ssh.Command("sed -i 's/native.cgroupdriver=cgroupfs/native.cgroupdriver=systemd/g' /usr/lib/systemd/system/docker.service"),
		mkdir("/etc/docker"),
		writeToFile(file, daemonJson),
		mkdir("/etc/systemd/system/docker.service.d"),
		ssh.Command("systemctl daemon-reload"),
		ssh.Command("systemctl restart docker"),
		ssh.Command("systemctl enable docker"),
	)

	return br.Err()
}

func InstallKubernetes(cfg CmdConfig, args map[string]string) error {
	// read in k8s.conf
	k8sConf, err := asset.Assets.Open("/etc/sysctl.d/k8s.conf")
	if err != nil {
		glog.Errorf("error reading /etc/sysctl.d/k8s.conf: %q", err)
		return err
	}
	defer k8sConf.Close()

	// selinux disable
	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)
	br.Run(
		ssh.Command("if [ $(getenforce) != 'Disabled' ]; then setenforce 0; fi"),
		ssh.Command("sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config"),
		ssh.Command("swapoff -a"),
		ssh.Command("if grep -Pq '^/dev/mapper/centos-swap' /etc/fstab; then "+
			"sed -ri.bak-$(date +%Y%m%dT%H%M%S) 's/(.*centos-swap.*)/#\\1/' /etc/fstab; fi"),
		yum("conntrack"),
		ssh.Command("modprobe br_netfilter"),
	)

	// run kubernetes install commands
	// for creates we use the version from machine status,
	// that is set as soon as machine starts being created
	bootstrapRepoUrl := fmt.Sprintf("http://%s:%s", cfg.templateData.BootstrapIp, cfg.templateData.BootstrapPort)
	k8sVersion := cfg.machineInstance.Status.KubernetesVersion
	br.Run(
		ssh.Cmd{Command: "cat - > /etc/sysctl.d/k8s.conf", Stdin: k8sConf},
		ssh.Command("sysctl --system"),
		yum("kubelet-"+k8sVersion),
		yum("kubectl-"+k8sVersion),
		yum("kubeadm-"+k8sVersion),
		ssh.Command("sed -i 's/cgroup-driver=cgroupfs/cgroup-driver=systemd/g' "+
			"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"),
		mkdir("/etc/kubernetes/"),
		ssh.Command("wget --output-document=/etc/kubernetes/kube-flannel.yml "+bootstrapRepoUrl+"/download/kube-flannel.yml"),
		ssh.Command("systemctl daemon-reload"),
		ssh.Command("systemctl enable kubelet"),
		ssh.Command("systemctl restart kubelet"),
	)

	return br.Err()
}

func KubeadmInit(cfg CmdConfig, args map[string]string) error {
	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)

	// kubeadm init
	br.Run(
		ssh.Command("kubeadm init --pod-network-cidr="+
			cfg.clusterInstance.Spec.ClusterNetwork.Pods.CIDRBlock+
			" --kubernetes-version="+cfg.clusterInstance.Spec.KubernetesVersion),
		ssh.Command("kubectl --kubeconfig /etc/kubernetes/admin.conf apply -f /etc/kubernetes/kube-flannel.yml --force=true"),
	)

	// get the lowercased hostname for /etc/host
	var buf bytes.Buffer
	cmd := ssh.Cmd{Command: "hostname", Stdout: &buf}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}
	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	err := util.Retry(120, 10*time.Second, func() error {
		for k, v := range cfg.machineInstance.Spec.Labels {
			c := ssh.Command("kubectl --kubeconfig /etc/kubernetes/kubelet.conf label node " + string(hostname) + " " + k + "=" + v)
			if err := c.Run(cfg.sshClient); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func KubeadmTokenCreate(cfg CmdConfig, args map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	cmd := ssh.Cmd{
		Command: "kubeadm token create --description " + cfg.machineInstance.GetName(),
		Stdout:  &buf,
	}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func KubeadmJoin(cfg CmdConfig, args map[string]string) error {
	token := args["token"]
	master := args["master"] + ":" + common.ApiEnpointPort

	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)

	// kubeadm join
	br.Run(
		ssh.Command("kubeadm join --token " + token +
			" --ignore-preflight-errors=all " +
			"--discovery-token-unsafe-skip-ca-verification " + master),
	)
	if br.Err() != nil {
		return br.Err()
	}

	// get the lowercased hostname for /etc/host
	var buf bytes.Buffer
	cmd := ssh.Cmd{Command: "hostname", Stdout: &buf}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}
	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	return util.Retry(120, 10*time.Second, func() error {
		for k, v := range cfg.machineInstance.Spec.Labels {
			err := ssh.Command("kubectl --kubeconfig /etc/kubernetes/kubelet.conf label node --overwrite " +
				string(hostname) + " " + k + "=" + v).Run(cfg.sshClient)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func GetKubeConfig(cfg CmdConfig, args map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	cmd := ssh.Cmd{
		Command: "cat /etc/kubernetes/admin.conf",
		Stdout:  &buf,
	}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeleteNode(cfg CmdConfig, args map[string]string) error {
	if err := runIfExists(cfg, "kubeadm", []string{"reset", "--force"}); err != nil {
		if err := runIfExists(cfg, "kubeadm", []string{"reset"}); err != nil {
			return err
		}
	}

	rpms := []string{
		"wget",
		"kubeadm",
		"kubelet",
		"kubernetes-cni",
		"kubectl",
		"docker",
		"docker-client",
		"docker-common",
		"nginx",
		"conntrack",
	}
	for _, rpm := range rpms {
		err := yumCheckAndRemove(cfg, rpm)
		if err != nil {
			return err
		}
	}

	// delete bootstrap repo file
	glog.Infof("deleting repo file for %s", cfg.machineInstance.GetName())
	if err := rmrf("/etc/yum.repos.d/sds_bootstrap.repo").Run(cfg.sshClient); err != nil {
		return err
	}

	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)

	// delete folders
	glog.Infof("deleting folders for %s", cfg.machineInstance.GetName())
	br.Run(
		rmrf("/etc/cni"),
		rmrf("/etc/docker"),
		rmrf("/etc/nginx"),
		rmrf("/etc/sysconfig/docker"),
		rmrf("/etc/ethertypes"),
		rmrf("/etc/kubernetes"),
		rmrf("/etc/systemd/system/kubelet.service.d"),
		rmrf("/var/lib/cni"),
		rmrf("/var/lib/docker"),
		rmrf("/var/lib/dockershim"),
		rmrf("/var/lib/etcd"),
		rmrf("/var/lib/etcd2"),
		rmrf("/var/lib/kubelet"),
		ssh.Command("ip link set cni0 down"),
		ssh.Command("ip link set flannel.1 down"),
		ssh.Command("ip link set docker0 down"),
		ssh.Command("ip link delete cni0"),
		ssh.Command("ip link delete flannel.1"),
		ssh.Command("ip link delete docker0"),
	)

	return br.Err()
}

func UpgradeMaster(cfg CmdConfig, args map[string]string) error {

	// for updates we use the version from cluster, as that is what we are upgrading to
	k8sVersion := cfg.clusterInstance.Spec.KubernetesVersion

	// get the lowercased hostname
	var buf bytes.Buffer
	cmd := ssh.Cmd{
		Command: "hostname",
		Stdout:  &buf,
	}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}
	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)
	br.Run(
		yum("kubeadm-"+k8sVersion, disableExclude("kubernetes")),
		ssh.Command("kubeadm upgrade apply v"+k8sVersion+" --feature-gates=CoreDNS=false -y"),
		ssh.Command("kubectl drain "+string(hostname)+" --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"),
		yum("kubelet-"+k8sVersion, disableExclude("kubernetes")),
		yum("kubectl-"+k8sVersion),
		ssh.Command("echo -n 'KUBELET_EXTRA_ARGS=--cgroup-driver=systemd' > /etc/sysconfig/kubelet"),
		ssh.Command("systemctl daemon-reload"),
		ssh.Command("systemctl restart kubelet"),
		ssh.Command("kubectl uncordon "+string(hostname)+" --kubeconfig=/etc/kubernetes/admin.conf"),
	)

	return br.Err()
}

func UpgradeNode(cfg CmdConfig, args map[string]string) error {
	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)

	// for updates we use the version from cluster, as that is what we are upgrading to
	k8sVersion := cfg.clusterInstance.Spec.KubernetesVersion

	// get the lowercased hostname for /etc/host
	var buf bytes.Buffer
	cmd := ssh.Cmd{Command: "hostname", Stdout: &buf}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}
	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	br.Run(
		ssh.Cmd{Command: "cat - > /etc/kubernetes/admin.conf",
			Stdin: strings.NewReader(args["admin.conf"])},
		ssh.Command("kubectl drain "+
			string(hostname)+" --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"),
		yum("kubeadm-"+k8sVersion, disableExclude("kubernetes")),
		yum("kubelet-"+k8sVersion, disableExclude("kubernetes")),
		yum("kubectl-"+k8sVersion),
	)
	if br.Err() != nil {
		return br.Err()
	}

	// intermittent failures are possible when pulling config from master
	err := util.Retry(60, 2*time.Second, func() error {
		return ssh.Command("kubeadm upgrade node config --kubelet-version $(kubelet --version | cut -d ' ' -f 2)").Run(cfg.sshClient)
	})
	if err != nil {
		return err
	}

	// configure kubelet
	br.Run(
		ssh.Command("echo -n 'KUBELET_EXTRA_ARGS=--cgroup-driver=systemd' > /etc/sysconfig/kubelet"),
	)

	br.Run(
		ssh.Command("systemctl daemon-reload"),
		ssh.Command("systemctl restart kubelet"),
		ssh.Command("kubectl uncordon "+string(hostname)+" --kubeconfig=/etc/kubernetes/admin.conf"),
	)

	return br.Err()
}

func DrainAndDeleteNode(cfg CmdConfig, args map[string]string) error {
	// get the lowercased hostname for /etc/host
	var buf bytes.Buffer
	cmd := ssh.Cmd{Command: "hostname", Stdout: &buf}
	if err := cmd.Run(cfg.sshClient); err != nil {
		return err
	}
	hostname := bytes.ToLower(bytes.TrimSpace(buf.Bytes()))

	// install the new kubeadm and run upgrade
	// install new kubelet and kubectl
	br := ssh.NewBatchRunner(cfg.sshClient, os.Stdout)
	br.Run(
		ssh.Cmd{Command: "cat - > /etc/kubernetes/admin.conf",
			Stdin: bytes.NewReader([]byte(args["admin.conf"]))},
		ssh.Command("kubectl drain "+
			string(hostname)+" --ignore-daemonsets --kubeconfig=/etc/kubernetes/admin.conf"),
		ssh.Command("kubectl delete node "+string(hostname)+
			" --kubeconfig=/etc/kubernetes/admin.conf"),
	)

	return br.Err()
}

type yumOption func(*yumConfig)

type yumConfig struct {
	action         string
	enableRepo     string
	disableRepo    string
	disableExclude string
}

const (
	yumDefaultRepo   = "sds_bootstrap"
	yumInstall       = "install"
	yumRemove        = "remove"
	yumListInstalled = "list installed"
)

func (y yumConfig) String() string {
	flags := []string{y.action}
	if y.disableRepo != "" {
		flags = append(flags, fmt.Sprintf("--disablerepo=%q", y.disableRepo))
	}
	if y.disableExclude != "" {
		flags = append(flags, fmt.Sprintf("--disableexcludes=%q", y.disableExclude))
	}
	if y.enableRepo != "" {
		flags = append(flags, fmt.Sprintf("--enablerepo=%q", y.enableRepo))
	}
	return strings.Join(flags, " ")
}

func remove(y *yumConfig) {
	y.action = yumRemove
}

func listInstalled(y *yumConfig) {
	y.action = yumListInstalled
}

func disableExclude(repo string) yumOption {
	return func(y *yumConfig) {
		y.disableExclude = repo
	}
}

// yum calls yum install on the rpm specified by default. you can use the
// yumOption funcs to change the call that is made.
func yum(rpm string, opts ...yumOption) ssh.Cmd {
	y := yumConfig{
		action:      "install",
		disableRepo: "*",
		enableRepo:  yumDefaultRepo,
	}

	for _, opt := range opts {
		opt(&y)
	}

	return ssh.Command(fmt.Sprintf("yum %s -y %s", y, rpm))
}

func yumCheckAndRemove(cfg CmdConfig, rpm string) error {
	// check if wget is installed, uninstall
	glog.Infof("checking %s for %s", rpm, cfg.machineInstance.GetName())
	err := yum(rpm, listInstalled).Run(cfg.sshClient)
	if err == nil {
		glog.Infof("deleting %s for %s", rpm, cfg.machineInstance.GetName())
		if err := yum(rpm, remove).Run(cfg.sshClient); err != nil {
			return err
		}
	}
	return nil
}

func writeToFile(file string, r io.Reader) ssh.Cmd {
	return ssh.Cmd{
		Command: fmt.Sprintf("cat - > %q", file),
		Stdin:   r,
	}
}

func mkdir(path string) ssh.Cmd {
	return ssh.Cmd{
		Command: fmt.Sprintf("mkdir -p %s", path),
	}
}

func rmrf(path string) ssh.Cmd {
	return ssh.Cmd{
		Command: fmt.Sprintf("rm -rf %s", path),
	}
}

// runIfExists if the command exists on the system it runs the command with the
// given args and returns any error. If the command does not exists it returns
// nil.
func runIfExists(cfg CmdConfig, cmd string, args []string) error {
	if err := ssh.Command("command -v " + cmd).Run(cfg.sshClient); err != nil {
		return nil
	}
	return ssh.Command(cmd + " " + strings.Join(args, " ")).Run(cfg.sshClient)
}
