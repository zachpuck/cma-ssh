// Generate a self-signed X.509 certificate for a TLS server. Outputs to
// 'cert.pem' and 'key.pem' and will overwrite existing files.
package cert

import (
	"archive/tar"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

const (
	rsaBits = 2048
)

func PemEncoded(cert, parent *x509.Certificate, key, priv crypto.Signer, outcert, outkey io.Writer) error {
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, parent, key.Public(), priv)
	if err != nil {
		return errors.Wrap(err, "Failed to create certificate")
	}

	if err := pem.Encode(outcert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return errors.Wrap(err, "failed to write data to cert.pem")
	}

	if err := pem.Encode(outkey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))}); err != nil {
		return errors.Wrap(err, "failed to write data to key.pem")
	}
	return nil
}

func NewCABundle() (*CABundle, error) {
	rootCA, err := FromCATemplate("samsung-cnct")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate root ca")
	}
	rootCAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "could not create root ca key")
	}

	k8sCA, err := FromCATemplate("kubernetes-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate kubernetes-ca")
	}
	k8sCAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "could not create k8s ca key")
	}

	etcdCA, err := FromCATemplate("etcd-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate etcd-ca")
	}
	etcdCAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "could not create etcd ca key")
	}

	k8sFrontProxyCA, err := FromCATemplate("kubernetes-front-proxy-ca")
	if err != nil {
		return nil, errors.Wrap(err, "could not generate kubernetes-front-proxy-ca")
	}
	k8sFrontProxyCAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes front proxy ca key")
	}

	kubeconfig, err := FromCertTemplate("kubernetes-admin", []string{"system:masters"}, nil, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubeconfig client cert")
	}
	kubeconfigKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubeconfig client key")
	}

	// https://github.com/kelseyhightower/kubernetes-the-hard-way/blob/master/docs/04-certificate-authority.md
	// https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#custom-certificates
	// https://kubernetes.io/docs/setup/certificates/#configure-certificates-manually
	return &CABundle{
		Root:          rootCA,
		RootKey:       rootCAKey,
		K8s:           k8sCA,
		K8sKey:        k8sCAKey,
		Etcd:          etcdCA,
		EtcdKey:       etcdCAKey,
		FrontProxy:    k8sFrontProxyCA,
		FrontProxyKey: k8sFrontProxyCAKey,
		Kubeconfig:    kubeconfig,
		KubeconfigKey: kubeconfigKey,
	}, nil
}

type CABundle struct {
	Root, K8s, Etcd, FrontProxy, Kubeconfig                *x509.Certificate
	RootKey, K8sKey, EtcdKey, FrontProxyKey, KubeconfigKey *rsa.PrivateKey
}

func (c CABundle) ToTar() (string, error) {
	var rootCAPem, rootCAKeyPem bytes.Buffer
	if err := PemEncoded(c.Root, c.Root, c.RootKey, c.RootKey, &rootCAPem, &rootCAKeyPem); err != nil {
		return "", errors.Wrap(err, "could not encode pem for root ca cert and key")
	}

	var k8sCAPem, k8sCAKeyPem bytes.Buffer
	if err := PemEncoded(c.K8s, c.Root, c.K8sKey, c.RootKey, &k8sCAPem, &k8sCAKeyPem); err != nil {
		return "", errors.Wrap(err, "could not encode pem for k8s ca cert and key")
	}

	var etcdCAPem, etcdCAKeyPem bytes.Buffer
	if err := PemEncoded(c.Etcd, c.Root, c.EtcdKey, c.RootKey, &etcdCAPem, &etcdCAKeyPem); err != nil {
		return "", errors.Wrap(err, "could not encode pem for etcd ca cert and key")
	}

	var k8sFrontProxyCAPem, k8sFrontProxyCAKeyPem bytes.Buffer
	if err := PemEncoded(c.FrontProxy, c.Root, c.FrontProxyKey, c.RootKey, &k8sFrontProxyCAPem, &k8sFrontProxyCAKeyPem); err != nil {
		return "", errors.Wrap(err, "could not encode pem for kubernetes front proxy ca cert and key")
	}

	var tarball bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &tarball)
	tw := tar.NewWriter(enc)
	var files = []struct {
		Name string
		Body []byte
		Mode int64
	}{
		{Name: "etcd/ca.crt", Body: etcdCAPem.Bytes(), Mode: 0644},
		{Name: "etcd/ca.key", Body: etcdCAKeyPem.Bytes(), Mode: 0600},
		{Name: "ca.crt", Body: k8sCAPem.Bytes(), Mode: 0644},
		{Name: "ca.key", Body: k8sCAKeyPem.Bytes(), Mode: 0600},
		{Name: "front-proxy-ca.crt", Body: k8sFrontProxyCAPem.Bytes(), Mode: 0644},
		{Name: "front-proxy-ca.key", Body: k8sFrontProxyCAKeyPem.Bytes(), Mode: 0600},
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
