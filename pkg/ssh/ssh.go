package ssh

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
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
		if c.err != nil {
			return c.currentCmd, c.err
		}
		c.currentCmd, c.err = c.exec(client, cmd)
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
	err = session.Run(cmd.Cmd)
	if err != nil {
		return cmd.Cmd, err
	}
	return cmd.Cmd, nil
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
	return bytes.TrimSpace(out), cmd.Cmd, nil
}
