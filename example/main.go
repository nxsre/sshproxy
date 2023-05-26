package main

import (
	"crypto/tls"
	"errors"
	"github.com/Shopify/sarama"
	logkafka "github.com/nxsre/logrus-kafka-hook"
	"github.com/nxsre/sshproxy"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func main() {
	authMethods := sshproxy.ReadPrivateKeys(sshKeys...)
	if len(authMethods) == 0 {
		log.Println("no SSH keys found")
	}

	hostKeyCallback, err := knownhosts.New(knownHosts)
	if err != nil {
		log.Fatal(err)
	}

	p := sshproxy.NewProxy(ssh.ClientConfig{
		Timeout:         10 * time.Second,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	})

	px := p.GetClient(sshproxy.Key{
		Host:     "10.211.x.x",
		Port:     22,
		Username: "root",
		//Password: "xxxxxx",
	})

	// use SimpleProducer to create an AsyncProducer
	producer, _ := SimpleProducer([]string{"192.168.10.4:32293", "192.168.10.4:32193", "192.168.10.4:32093"}, sarama.CompressionSnappy, sarama.WaitForLocal, nil, px)

	// use DefaultFormatter to create a JSONFormatter with pre-defined fields or override with any fields
	// create the Hook and use the builder functions to apply configurations
	hook := logkafka.New().WithFormatter(logkafka.DefaultFormatter(logrus.Fields{"type": "myappName"})).WithProducer(producer).WithTopic("test")

	// create a new logger and add the hook
	log := logrus.New()
	log.Hooks.Add(hook)
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(io.Discard)

	for i := 0; i <= 100; i++ {
		log.Debug("Hello World ", i)
		time.Sleep(500 * time.Millisecond)
	}
}

const flushFrequency = 500 * time.Millisecond

// ErrNoProducer is used when hook Fire method is called and no producer was configured.
var ErrNoProducer = errors.New("no producer defined")

// SimpleProducer accepts a minimal set of configurations and creates an AsyncProducer.
func SimpleProducer(brokers []string, compression sarama.CompressionCodec, ack sarama.RequiredAcks, tlscfg *tls.Config, px proxy.Dialer) (sarama.AsyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = ack
	cfg.Producer.Compression = compression
	cfg.Producer.Flush.Frequency = flushFrequency

	if tlscfg != nil {
		cfg.Net.TLS.Enable = true
		cfg.Net.TLS.Config = tlscfg
	}

	if px != nil {
		cfg.Net.Proxy = struct {
			Enable bool
			Dialer proxy.Dialer
		}{Enable: true, Dialer: px}
	}
	return sarama.NewAsyncProducer(brokers, cfg)
}

var home = func() string {
	user, err := user.Current()
	if err == nil && user.HomeDir != "" {
		return user.HomeDir
	}

	return os.Getenv("HOME")
}()

var (
	sshKeyDir = envStr("HOS_KEY_DIR", filepath.Join(home, ".ssh"))
	sshKeys   = []string{
		filepath.Join(sshKeyDir, "id_rsa"),
		filepath.Join(sshKeyDir, "id_ed25519"),
	}
	knownHosts = filepath.Join(sshKeyDir, "known_hosts")
)

func envStr(name, fallback string) string {
	if s := os.Getenv(name); s != "" {
		return s
	}
	return fallback
}
