package sshproxy

import (
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

// Reads a SSH private key file.
func getKeyFile(path string) (ssh.Signer, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(buf)
}

func ReadPrivateKeys(paths ...string) (methods []ssh.AuthMethod) {
	for _, path := range paths {
		if signer, err := getKeyFile(path); err == nil {
			log.Println("loaded private key", path)
			methods = append(methods, ssh.PublicKeys(signer))
		} else {
			log.Printf("unable to load private key %q: %v", path, err)
		}
	}
	return methods
}
