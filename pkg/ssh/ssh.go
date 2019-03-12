package ssh

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// Client contains the underlying net.Conn and an ssh.Client for the conn.
type Client struct {
	c net.Conn
	*ssh.Client
}

func (c *Client) Close() error {
	return c.Client.Close()
}

// MachineParams contains parameters for individual machines.
type MachineParams struct {
	Username   string
	Host       string
	PublicHost string
	Port       int32
	Password   string
}

// ClusterParams contains parameters for the entire cluster.
type ClusterParams struct {
	Name              string
	PrivateKey        string // These are base64 _and_ PEM encoded Eliptic
	PublicKey         string // Curve (EC) keys used in JSON and YAML.
	K8SVersion        string
	ControlPlaneNodes []MachineParams
	WorkerNodes       []MachineParams
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

// BatchRunner is used for running multiple ssh commands on a single client.
type BatchRunner struct {
	client *Client
	out    io.Writer
	err    error
}

func NewBatchRunner(c *Client, out io.Writer) *BatchRunner {
	return &BatchRunner{client: c, out: out}
}

func (b *BatchRunner) Err() error {
	return b.err
}

// Run executes all the commands on the client. If any of the commands
// return an error then all remaining commands will be skipped. You can check
// the error with BatchRunner.Err(). Stdout and Stderr from the command are
// interleaved and are written to the BatchRunners output.
func (b *BatchRunner) Run(cmds ...Cmd) {
	ocw := combinedWriter{w: b.out}
	for _, cmd := range cmds {
		// do not run commands if there is any previous error
		if b.err != nil {
			return
		}
		cmd.Stdout = &ocw
		cmd.Stderr = &ocw
		b.err = cmd.Run(b.client)
	}
}

type combinedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (m *combinedWriter) Write(b []byte) (int, error) {
	m.mu.Lock()
	n, err := m.w.Write(b)
	m.mu.Unlock()
	return n, err
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
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(remoteCmd); err != nil {
		return err
	}
	fmt.Println(b.String())

	return nil
}

type Cmd struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Command is a convenience method to create Cmd.
func Command(command string) Cmd {
	return Cmd{
		Command: command,
	}
}

// CommandWithInput is a convenience method to create Cmd that takes input from
// a supplied io.Reader.
func CommandWithInput(command string, r io.Reader) Cmd {
	return Cmd{
		Command: command,
		Stdin:   r,
	}
}

// Run executes the command on the client. It handles creating ssh sessions.
func (c Cmd) Run(client *Client) error {
	s, err := client.Client.NewSession()
	if err != nil {
		return errors.Wrap(err, "could not create a new ssh session")
	}
	// We will close the session in case the code below changes for any reason.
	defer s.Close()

	s.Stdin = c.Stdin
	s.Stdout = c.Stdout
	s.Stderr = c.Stderr

	if err := s.Run(c.Command); err != nil {
		return ErrorCmd{
			cmd: c,
			err: err,
		}
	}
	return nil
}

// ErrorCmd is helper to return the error msg as well as the Cmd that was run.
type ErrorCmd struct {
	cmd Cmd
	err error
}

// Error implements the error interface
func (e ErrorCmd) Error() string {
	return e.err.Error()
}

// Cmd returns the command that was run.
func (e ErrorCmd) Cmd() string {
	return e.cmd.Command
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

func SetupPrivateKeyAccess(machine MachineParams, privateKey []byte, publicKey []byte) error {
	host := machine.PublicHost
	if host == "" {
		host = machine.Host
	}

	err := AddPublicKeyToRemoteNode(
		host,
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

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, machine.Port), config)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(testCmd); err != nil {
		return err
	}
	fmt.Println(b.String())

	return nil
}
