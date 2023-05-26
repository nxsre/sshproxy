package sshproxy

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	key        Key
	sshCert    *ssh.Certificate
	sshConfig  ssh.ClientConfig
	sshClient  *ssh.Client
	httpClient *http.Client
	mtx        sync.Mutex
}

// Key is used for reusing SSH connections.
type Key struct {
	Host     string
	Port     uint16
	Username string
	Password string
}

// hostPort returns the Host joined with the port.
func (key *Key) hostPort() string {
	return net.JoinHostPort(key.Host, strconv.Itoa(int(key.Port)))
}

func (key *Key) String() string {
	hp := key.hostPort()
	if key.Username == "" {
		return hp
	}
	return fmt.Sprintf("%s@%s", key.Username, hp)
}

// establishes the SSH connection and sets up the HTTP Client.
func (client *Client) connect() error {
	sshClient, err := ssh.Dial("tcp", client.key.hostPort(), &client.sshConfig)
	if err != nil {
		log.Printf("SSH connection to %s failed: %v", client.key.String(), err)
		return err
	}

	client.sshClient = sshClient
	log.Printf("SSH connection to %s established", client.key.String())

	return nil
}

// establishes a TCP connection through SSH.
func (client *Client) dial(network, address string) (net.Conn, error) {
	client.mtx.Lock()
	defer client.mtx.Unlock()

	retried := false

retry:
	if client.sshClient == nil {
		if err := client.connect(); err != nil {
			return nil, err
		}
	}

	conn, err := client.sshClient.Dial(network, address)

	if err != nil && !retried && (errors.Is(err, io.EOF) || !client.isAlive()) {
		// ssh connection broken
		client.sshClient.Close()
		client.sshClient = nil

		// Clean up idle HTTP connections
		client.httpClient.Transport.(*http.Transport).CloseIdleConnections()

		retried = true
		goto retry
	}

	if err == nil {
		log.Printf("TCP forwarding via %s to %s established", client.key.String(), address)
	} else {
		log.Printf("TCP forwarding via %s to %s failed: %s", client.key.String(), address, err)
	}

	return &ProxyConn{conn}, err
}

// checks if the SSH Client is still alive by sending a keep alive request.
func (client *Client) isAlive() bool {
	_, _, err := client.sshClient.Conn.SendRequest("keepalive@openssh.com", true, nil)
	return err == nil
}

func (client *Client) Dial(network, address string) (net.Conn, error) {
	return client.dial(network, address)
}

func (client *Client) HttpClient() *http.Client {
	return client.httpClient
}

type ProxyConn struct {
	net.Conn
}

func (*ProxyConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (*ProxyConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (*ProxyConn) SetWriteDeadline(t time.Time) error {
	return nil
}
