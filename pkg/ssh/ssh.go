package ssh

import (
	"io"
	"net"

	"golang.org/x/crypto/ssh"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Client contains the underlying net.Conn and an ssh.Client for the conn.
type Client struct {
	c net.Conn
	*ssh.Client
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
	})
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(conn, newCh, reqCh)
	return &Client{c: c, Client: client}, nil
}

type CommandRunner struct {
	Stdout io.Writer
	Stderr io.Writer
	err    error
}

type Command struct {
	Cmd   string
	Stdin io.Reader
}

func (c *CommandRunner) Run(client *ssh.Client, cmds ...Command) error {
	for _, cmd := range cmds {
		if c.err != nil {
			return c.err
		}
		c.err = c.exec(client, cmd)
	}
	return c.err
}

const proxy = "http://192.168.99.1:3128"

var proxyEnvs = []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY"}

func (c *CommandRunner) exec(client *ssh.Client, cmd Command) error {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("pkg/ssh: exec")

	log.Info("running command on remote host", cmd, client.RemoteAddr())
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	for _, proxyEnv := range proxyEnvs {
		if err := session.Setenv(proxyEnv, proxy); err != nil {
			log.Error(err, "make sure AcceptEnv is set for remote host in /etc/ssh/sshd_config", proxyEnv)
		}
	}

	session.Stdin = cmd.Stdin
	session.Stdout = c.Stdout
	session.Stderr = c.Stderr
	if err := session.Run(cmd.Cmd); err != nil {
		switch e := err.(type) {
		case *ssh.ExitMissingError:
			log.Info("comand exitted without status", e)
		case *ssh.ExitError:
			log.Info("command unsuccessful", e.String())
		}
		return err
	}
	return nil
}
