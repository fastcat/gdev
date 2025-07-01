package gocache_sftp

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"fastcat.org/go/gdev/addons/gocache"
)

type sftpCacheFactory struct{}

// Want implements gocache.RemoteStorageFactory.
func (sftpCacheFactory) Want(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return u.Scheme == "sftp"
}

// New implements gocache.RemoteStorageFactory.
func (s sftpCacheFactory) New(uri string) (_ gocache.ReadonlyStorageBackend, finalErr error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	kh, err := knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("failed to load known hosts: %w", err)
	}

	var agentConn net.Conn
	var sshClient *ssh.Client
	var sftpClient *sftp.Client
	defer func() {
		if finalErr != nil {
			if agentConn != nil {
				_ = agentConn.Close()
			}
			if sftpClient != nil {
				_ = sftpClient.Close()
			}
			if sshClient != nil {
				_ = sshClient.Close()
			}
		}
	}()

	// ssh-agent(1) provides a UNIX socket at $SSH_AUTH_SOCK.
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable is not set")
	}
	agentConn, err = net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent at %q: %w", socket, err)
	}
	sshAgent := agent.NewClient(agentConn)

	cfg := ssh.ClientConfig{
		Timeout:         5 * time.Second,
		HostKeyCallback: kh,
		User:            u.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(sshAgent.Signers),
		},
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "22"
	}

	sshClient, err = ssh.Dial("tcp", net.JoinHostPort(host, port), &cfg)
	if err != nil {
		return nil, err
	}
	sftpClient, err = sftp.NewClient(sshClient, sftp.UseConcurrentReads(false))
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(u.Path, "/") {
		return nil, fmt.Errorf("sftp base directory must be absolute, got %q", u.Path)
	}
	return gocache.DiskDirFromFS(&sftpStorageBackend{sshClient, sftpClient, u}), nil
}
