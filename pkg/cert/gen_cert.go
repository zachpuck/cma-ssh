// Generate a self-signed X.509 certificate for a TLS server. Outputs to
// 'cert.pem' and 'key.pem' and will overwrite existing files.
package cert

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

const (
	rsaBits = 2048
)

func NewCABundle() (*CABundle, error) {
	rootCA, err := FromCATemplate("samsung-cnct")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate root ca")
	}
	rootCAKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not create root ca key")
	}
	rootKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rootCAKey)})
	rootDer, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, rootCAKey.Public(), rootCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not create der bytes for root ca")
	}
	rootCA, err = x509.ParseCertificate(rootDer)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse certificate for root ca")
	}
	rootPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDer})

	k8sCA, err := FromCATemplate("kubernetes-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate kubernetes-ca")
	}
	k8sCAKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not create k8s ca key")
	}
	k8sKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k8sCAKey)})
	k8sDer, err := x509.CreateCertificate(rand.Reader, k8sCA, rootCA, k8sCAKey.Public(), rootCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not create der bytes for k8s ca")
	}
	k8sCA, err = x509.ParseCertificate(k8sDer)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse certificate for k8s ca")
	}
	k8sPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: k8sDer})

	etcdCA, err := FromCATemplate("etcd-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate etcd-ca")
	}
	etcdCAKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not create etcd ca key")
	}
	etcdKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(etcdCAKey)})
	etcdDer, err := x509.CreateCertificate(rand.Reader, etcdCA, rootCA, etcdCAKey.Public(), rootCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not create der bytes for etcd ca")
	}
	etcdPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: etcdDer})

	k8sFrontProxyCA, err := FromCATemplate("kubernetes-front-proxy-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate kubernetes-front-proxy-ca")
	}
	k8sFrontProxyCAKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes front proxy ca key")
	}
	k8sFrontProxyKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k8sFrontProxyCAKey)})
	k8sFrontProxyDer, err := x509.CreateCertificate(rand.Reader, k8sFrontProxyCA, rootCA, k8sFrontProxyCAKey.Public(), rootCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not create der bytes for k8s front proxy ca")
	}
	k8sFrontProxyPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: k8sFrontProxyDer})

	kubeconfig, err := FromCertTemplate("kubernetes-admin", []string{"system:masters"}, nil, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubeconfig client cert")
	}
	kubeconfigKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubeconfig client key")
	}
	kubeconfigKeyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(kubeconfigKey)})
	kubeconfigDer, err := x509.CreateCertificate(rand.Reader, kubeconfig, k8sCA, kubeconfigKey.Public(), k8sCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not write der bytes for kubeconfig")
	}
	kubeconfigPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kubeconfigDer})

	// https://github.com/kelseyhightower/kubernetes-the-hard-way/blob/master/docs/04-certificate-authority.md
	// https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#custom-certificates
	// https://kubernetes.io/docs/setup/certificates/#configure-certificates-manually
	return &CABundle{
		Root:          rootPem,
		RootKey:       rootKeyPem,
		K8s:           k8sPem,
		K8sKey:        k8sKeyPem,
		Etcd:          etcdPem,
		EtcdKey:       etcdKeyPem,
		FrontProxy:    k8sFrontProxyPem,
		FrontProxyKey: k8sFrontProxyKeyPem,
		K8sClient:    kubeconfigPem,
		K8sClientKey: kubeconfigKeyPem,
	}, nil
}

// CABundle is the pem encoded certs required for a kubernetes cluster
type CABundle struct {
	Root, RootKey             []byte
	K8s, K8sKey               []byte
	Etcd, EtcdKey             []byte
	FrontProxy, FrontProxyKey []byte
	K8sClient, K8sClientKey []byte
}

const (
	mapKeyRoot = "root.crt"
	mapKeyRootKey = "root.key"
	mapKeyK8s = "ca.crt"
	mapKeyK8sKey = "ca.key"
	mapKeyEtcd = "etcd.crt"
	mapKeyEtcdKey = "etcd.key"
	mapKeyFrontProxy = "front-proxy.crt"
	mapKeyFrontProxyKey = "front-proxy.key"
	mapKeyK8sClient = "k8s-client.crt"
	mapKeyK8sClientKey = "k8s-client.key"
)

func (c *CABundle) Set(key string, value []byte) {
	switch key {
	case mapKeyRoot:
		c.Root = value
	case mapKeyRootKey:
		c.RootKey = value
	case mapKeyK8s:
		c.K8s = value
	case mapKeyK8sKey:
		c.K8sKey = value
	case mapKeyEtcd:
		c.Etcd = value
	case mapKeyEtcdKey:
		c.EtcdKey = value
	case mapKeyFrontProxy:
		c.FrontProxy = value
	case mapKeyFrontProxyKey:
		c.FrontProxyKey = value
	case mapKeyK8sClient:
		c.K8sClient = value
	case mapKeyK8sClientKey:
		c.K8sClientKey = value
	}
}

func CABundleFromMap(m map[string][]byte) (*CABundle, error) {
	keys := []string{
		mapKeyRoot,
		mapKeyRootKey,
		mapKeyK8s,
		mapKeyK8sKey,
		mapKeyEtcd,
		mapKeyEtcdKey,
		mapKeyFrontProxy,
		mapKeyFrontProxyKey,
		mapKeyK8sClient,
		mapKeyK8sClientKey,
	}
	bundle := &CABundle{}
	for _, key := range keys {
		val, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in data", key)
		}
		bundle.Set(key, val)
	}
	return bundle, nil
}

func (c CABundle) MergeWithMap(m map[string][]byte) {
	m[mapKeyRoot] = c.Root
	m[mapKeyRootKey] = c.RootKey
	m[mapKeyK8s] = c.K8s
	m[mapKeyK8sKey] = c.K8sKey
	m[mapKeyEtcd] = c.Etcd
	m[mapKeyEtcdKey] = c.EtcdKey
	m[mapKeyFrontProxy] = c.FrontProxy
	m[mapKeyFrontProxyKey] = c.FrontProxyKey
	m[mapKeyK8sClient] = c.K8sClient
	m[mapKeyK8sClientKey] = c.K8sClientKey
}

func (c CABundle) ToTar() (string, error) {
	var tarball bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &tarball)
	tw := tar.NewWriter(enc)
	var files = []struct {
		Name string
		Body []byte
		Mode int64
	}{
		{Name: "etcd/ca.crt", Body: c.Etcd, Mode: 0644},
		{Name: "etcd/ca.key", Body: c.EtcdKey, Mode: 0600},
		{Name: "ca.crt", Body: c.K8s, Mode: 0644},
		{Name: "ca.key", Body: c.K8sKey, Mode: 0600},
		{Name: "front-proxy-ca.crt", Body: c.FrontProxy, Mode: 0644},
		{Name: "front-proxy-ca.key", Body: c.FrontProxyKey, Mode: 0600},
	}
	hdr := tar.Header{
		Name:     "etcd/",
		Mode:     0755,
		ModTime:  time.Now(),
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(&hdr); err != nil {
		return "", errors.Wrap(err, "could not write etcd dir tar header")
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name:    file.Name,
			Mode:    file.Mode,
			ModTime: time.Now(),
			Size:    int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return "", errors.Wrap(err, "could not write tar header")
		}
		if _, err := tw.Write(file.Body); err != nil {
			return "", errors.Wrap(err, "could not write tar body")
		}
	}
	if err := tw.Close(); err != nil {
		return "", errors.Wrap(err, "could not close tar writer")
	}
	if err := enc.Close(); err != nil {
		return "", errors.Wrap(err, "could not close b64 encoder")
	}

	return tarball.String(), nil
}

func (c *CABundle) Kubeconfig(name, apiserverAddress string) ([]byte, error) {
	clusterName := name
	userName := "kubernetes-admin"
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)
	config := &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   apiserverAddress,
				CertificateAuthorityData: c.K8s,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  name,
				AuthInfo: "kubernetes-admin",
			},
		},
		AuthInfos:      map[string]*api.AuthInfo{},
		CurrentContext: contextName,
	}

	config.AuthInfos[userName] = &api.AuthInfo{
		ClientKeyData: c.K8sClientKey,
		ClientCertificateData: c.K8sClient,
	}
	kubeconfig, err := runtime.Encode(latest.Codec, config)
	if err != nil {
		return nil, err
	}
	return kubeconfig, nil
}

func FromCATemplate(commonName string) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate serial number")
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Samsung CNCT"},
			CommonName:   commonName,
			Province:     []string{"Washington"},
			Locality:     []string{"Seattle"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		IsCA:      true,

		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	return template, nil

}

func FromCertTemplate(cn string, o []string, hosts []string, serverAuth, clientAuth bool) (*x509.Certificate, error) {
	if len(hosts) == 0 && serverAuth {
		return nil, fmt.Errorf("no hosts provided")
	}

	notBefore := time.Now()

	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate serial number")
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: o,
			CommonName:   cn,
			Province:     []string{"Washington"},
			Locality:     []string{"Seattle"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		IsCA:      true,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	if serverAuth {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}
	if clientAuth {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	return template, nil
}
