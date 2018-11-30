package ssh

import (
	"io"
	"net"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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

func (c *CommandRunner) GetOutput(client *ssh.Client, cmd Command) ([]byte, error) {
	return c.execWithOutput(client, cmd)
}


func (c *CommandRunner) exec(client *ssh.Client, cmd Command) error {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("ssh exec()")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			log.Error(err, "Could not close session after command", "command", cmd)
		}
	}(session)

	session.Stdin = cmd.Stdin
	session.Stdout = c.Stdout
	session.Stderr = c.Stderr
	err = session.Run(cmd.Cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *CommandRunner) execWithOutput(client *ssh.Client, cmd Command) ([]byte, error) {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("ssh execWithOutput()")

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			log.Error(err, "Could not close session after command", "command", cmd)
		}
	}(session)

	session.Stdin = cmd.Stdin
	session.Stderr = c.Stderr
	out, err := session.Output(cmd.Cmd)
	return out,nil
}