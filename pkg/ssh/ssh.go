package ssh

import (
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
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
	log.Printf("running %s on remote host %v", cmd, client.RemoteAddr())
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	for _, proxyEnv := range proxyEnvs {
		if err := session.Setenv(proxyEnv, proxy); err != nil {
			log.Printf("make sure AcceptEnv is set for %s in /etc/ssh/sshd_config: %v", proxyEnv, err)
		}
	}

	session.Stdin = cmd.Stdin
	session.Stdout = c.Stdout
	session.Stderr = c.Stderr
	if err := session.Run(cmd.Cmd); err != nil {
		switch e := err.(type) {
		case *ssh.ExitMissingError:
			log.Printf("comand exitted without status: %v", e)
		case *ssh.ExitError:
			log.Printf("command unsuccessful: %v", e.String())
		}
		return err
	}
	return nil
}
