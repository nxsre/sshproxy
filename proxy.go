package sshproxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Proxy holds the HTTP Client and the SSH connection pool.
type Proxy struct {
	clients   map[Key]*Client
	sshConfig ssh.ClientConfig
	mtx       sync.Mutex
}

// NewProxy creates a new proxy.
func NewProxy(config ssh.ClientConfig) *Proxy {
	return &Proxy{
		clients:   make(map[Key]*Client),
		sshConfig: config,
	}
}

// getClient returns a (un)connected SSH Client.
func (proxy *Proxy) GetClient(key Key) *Client {
	proxy.mtx.Lock()
	defer proxy.mtx.Unlock()

	// connection established?
	pClient := proxy.clients[key]
	if pClient != nil {
		return pClient
	}

	pClient = &Client{
		key:       key,
		sshConfig: proxy.sshConfig, // make copy
	}
	pClient.sshConfig.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := proxy.sshConfig.HostKeyCallback(hostname, remote, key); err != nil {
			return err
		}
		if cert, ok := key.(*ssh.Certificate); ok && cert != nil {
			pClient.sshCert = cert
		}
		return nil
	}

	if key.Username != "" {
		pClient.sshConfig.User = key.Username
	}

	if key.Username != "" {
		pClient.sshConfig.Auth = append(pClient.sshConfig.Auth, ssh.Password(key.Password))
	}

	pClient.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return pClient.dial(network, addr)
			},
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return nil, errors.New("not implemented")
			},
		},
	}

	// set and return the new connection
	proxy.clients[key] = pClient
	return pClient
}
