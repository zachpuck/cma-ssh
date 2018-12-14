package ssh

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"path/filepath"
	"time"
)

// Client contains the underlying net.Conn and an ssh.Client for the conn.
type Client struct {
	c net.Conn
	*ssh.Client
}

type SSHMachineParams struct {
	Username string
	Host     string
	Port     int32
	Password string
}

type SSHClusterParams struct {
	Name              string
	PrivateKey        string // These are base64 _and_ PEM encoded Eliptic
	PublicKey         string // Curve (EC) keys used in JSON and YAML.
	K8SVersion        string
	ControlPlaneNodes []SSHMachineParams
	WorkerNodes       []SSHMachineParams
}

// NewClient returns a client with the underlying net.Conn and an ssh.Client.
// You must close the ssh.Client and the net.Conn.
func NewClient(address, user string, privateKey []byte) (*Client, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	c, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	conn, newCh, reqCh, err := ssh.NewClientConn(c, address, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(conn, newCh, reqCh)
	return &Client{c: c, Client: client}, nil
}

type CommandRunner struct {
	Stdout     io.Writer
	Stderr     io.Writer
	err        error
	currentCmd string
}

type Command struct {
	Cmd   string
	Stdin io.Reader
}

func (c *CommandRunner) Run(client *ssh.Client, cmds ...Command) (string, error) {
	for _, cmd := range cmds {
		c.currentCmd, c.err = c.exec(client, cmd)
		if c.err != nil {
			return c.currentCmd, c.err
		}
	}
	return c.currentCmd, c.err
}

func (c *CommandRunner) GetOutput(client *ssh.Client, cmd Command) ([]byte, string, error) {
	return c.execWithOutput(client, cmd)
}

func (c *CommandRunner) exec(client *ssh.Client, cmd Command) (string, error) {

	session, err := client.NewSession()
	if err != nil {
		return cmd.Cmd, err
	}
	defer session.Close()

	session.Stdin = cmd.Stdin
	session.Stdout = c.Stdout
	session.Stderr = c.Stderr
	return cmd.Cmd, session.Run(cmd.Cmd)
}

func (c *CommandRunner) execWithOutput(client *ssh.Client, cmd Command) ([]byte, string, error) {

	session, err := client.NewSession()
	if err != nil {
		return nil, cmd.Cmd, err
	}
	defer session.Close()

	session.Stdin = cmd.Stdin
	session.Stderr = c.Stderr
	out, err := session.Output(cmd.Cmd)
	return bytes.TrimSpace(out), cmd.Cmd, err
}

// AddPublicKeyToRemoteNode will add the publicKey to the username@host:port's authorized_keys file w/password
func AddPublicKeyToRemoteNode(host string, port int32, username string, password string, publicKey []byte) error {
	var sshDir = filepath.Join("${HOME}", ".ssh")
	var authorizedKeysFile = filepath.Join(sshDir, "authorized_keys")

	remoteCmd := fmt.Sprintf("mkdir -p %s && chmod 700 %s && echo %s >> %s && chmod 600 %s",
		sshDir,
		sshDir,
		bytes.TrimSuffix(publicKey, []byte("\n")),
		authorizedKeysFile,
		authorizedKeysFile)

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(remoteCmd); err != nil {
		return err
	}
	fmt.Println(b.String())

	return nil
}

// GenerateSSHKeyPair creates a ECDSA a x509 ASN.1-DER format-PEM encoded private
// key string and a SHA256 encoded public key string
func GenerateSSHKeyPair() (private []byte, public []byte, err error) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// validate private key
	publicKey := &privateKey.PublicKey
	if !curve.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, nil, err
	}

	// convert to x509 ASN.1, DER format
	privateDERBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}

	// generate pem encoded private key
	privatePEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateDERBytes,
	})

	// generate public key fingerprint
	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)

	return privatePEMBytes, pubKeyBytes, nil
}

func SetupPrivateKeyAccess(machine SSHMachineParams, privateKey []byte, publicKey []byte) error {
	err := AddPublicKeyToRemoteNode(
		machine.Host,
		machine.Port,
		machine.Username,
		machine.Password,
		publicKey)
	if err != nil {
		return err
	}

	// Test private key
	testCmd := "echo cma-ssh: $(date) >> ~/.ssh/test-pvka"

	key, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            machine.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", machine.Host, machine.Port), config)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(testCmd); err != nil {
		return err
	}
	fmt.Println(b.String())

	return nil
}
