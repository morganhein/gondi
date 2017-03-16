package devices

import (
	"io/ioutil"

	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func CreateSSHConfig(options schema.ConnectOptions) (sshConfig *ssh.ClientConfig) {
	sshConfig = &ssh.ClientConfig{
		User: options.Username,
	}
	if options.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(options.Password),
		}
	}
	if options.Cert != "" {
		sshConfig.Auth = []ssh.AuthMethod{
			publicKeyFile(options.Cert),
		}
	}
	return
}
