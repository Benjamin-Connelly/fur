package remote

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/pkg/sftp"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ConnState represents the connection lifecycle state.
type ConnState int

const (
	ConnDisconnected ConnState = iota
	ConnConnecting
	ConnConnected
	ConnReconnecting
)

// String returns a human-readable state label.
func (s ConnState) String() string {
	switch s {
	case ConnConnecting:
		return "Connecting"
	case ConnConnected:
		return "Connected"
	case ConnReconnecting:
		return "Reconnecting"
	default:
		return "Disconnected"
	}
}

// Conn manages an SSH connection and SFTP client to a remote host.
type Conn struct {
	target    Target
	sshClient *ssh.Client
	sftp      *sftp.Client
	sshCfg   *ssh_config.Config // parsed ~/.ssh/config

	state     ConnState
	lastError error
	mu        sync.RWMutex

	done      chan struct{}
	keepalive time.Duration
}

// NewConn creates a new remote connection manager.
func NewConn(target Target) *Conn {
	return &Conn{
		target:    target,
		sshCfg:   loadSSHConfig(),
		done:      make(chan struct{}),
		keepalive: 30 * time.Second,
	}
}

// loadSSHConfig reads and parses ~/.ssh/config. Returns nil on failure.
func loadSSHConfig() *ssh_config.Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	f, err := os.Open(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		return nil
	}
	defer f.Close()
	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil
	}
	return cfg
}

// configGet looks up a key for the target host in the parsed SSH config.
// Returns empty string if config is nil or key not found.
func (c *Conn) configGet(key string) string {
	if c.sshCfg == nil {
		return ""
	}
	val, err := c.sshCfg.Get(c.target.Host, key)
	if err != nil {
		return ""
	}
	return val
}

// Connect establishes the SSH and SFTP connections.
func (c *Conn) Connect() error {
	c.mu.Lock()
	c.state = ConnConnecting
	c.mu.Unlock()

	config, err := c.buildSSHConfig()
	if err != nil {
		c.mu.Lock()
		c.state = ConnDisconnected
		c.lastError = err
		c.mu.Unlock()
		return fmt.Errorf("SSH config: %w", err)
	}

	host := c.resolveHost()
	port := c.resolvePort()
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		c.mu.Lock()
		c.state = ConnDisconnected
		c.lastError = err
		c.mu.Unlock()
		return fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		c.mu.Lock()
		c.state = ConnDisconnected
		c.lastError = err
		c.mu.Unlock()
		return fmt.Errorf("SFTP session: %w", err)
	}

	c.mu.Lock()
	c.sshClient = sshClient
	c.sftp = sftpClient
	c.state = ConnConnected
	c.lastError = nil
	c.mu.Unlock()

	// Resolve ~ in remote path using SFTP home directory
	if err := c.resolveRemotePath(); err != nil {
		sftpClient.Close()
		sshClient.Close()
		c.mu.Lock()
		c.state = ConnDisconnected
		c.lastError = err
		c.mu.Unlock()
		return fmt.Errorf("resolving remote path: %w", err)
	}

	go c.keepaliveLoop()

	return nil
}

// resolveRemotePath expands ~ and relative paths using SFTP RealPath.
func (c *Conn) resolveRemotePath() error {
	path := c.target.Path
	if path == "" || path == "." {
		// Default to home directory
		home, err := c.sftp.RealPath(".")
		if err != nil {
			return fmt.Errorf("getting remote home: %w", err)
		}
		c.target.Path = home
		return nil
	}

	if len(path) >= 2 && path[0] == '~' && path[1] == '/' {
		// ~/foo → /home/user/foo
		home, err := c.sftp.RealPath(".")
		if err != nil {
			return fmt.Errorf("getting remote home: %w", err)
		}
		c.target.Path = home + path[1:]
		return nil
	}

	if path == "~" {
		home, err := c.sftp.RealPath(".")
		if err != nil {
			return fmt.Errorf("getting remote home: %w", err)
		}
		c.target.Path = home
		return nil
	}

	// Absolute paths stay as-is
	return nil
}

// Close shuts down the connection.
func (c *Conn) Close() error {
	select {
	case <-c.done:
		// already closed
	default:
		close(c.done)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	if c.sftp != nil {
		if err := c.sftp.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.sftp = nil
	}
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.sshClient = nil
	}
	c.state = ConnDisconnected
	return firstErr
}

// SFTP returns the SFTP client. Returns nil if not connected.
func (c *Conn) SFTP() *sftp.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sftp
}

// State returns the current connection state.
func (c *Conn) State() ConnState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// LastError returns the most recent connection error.
func (c *Conn) LastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// Target returns the connection target.
func (c *Conn) Target() Target {
	return c.target
}

// Reconnect attempts to re-establish the connection with exponential backoff.
func (c *Conn) Reconnect() error {
	c.mu.Lock()
	c.state = ConnReconnecting
	c.mu.Unlock()

	// Close existing connection
	c.mu.Lock()
	if c.sftp != nil {
		c.sftp.Close()
		c.sftp = nil
	}
	if c.sshClient != nil {
		c.sshClient.Close()
		c.sshClient = nil
	}
	c.mu.Unlock()

	// Recreate done channel for new keepalive loop
	c.done = make(chan struct{})

	backoff := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		15 * time.Second,
		30 * time.Second,
	}

	for i, delay := range backoff {
		if err := c.Connect(); err == nil {
			return nil
		}

		if i < len(backoff)-1 {
			select {
			case <-time.After(delay):
			case <-c.done:
				return fmt.Errorf("reconnection cancelled")
			}
		}
	}

	return fmt.Errorf("reconnection failed after %d attempts", len(backoff))
}

func (c *Conn) buildSSHConfig() (*ssh.ClientConfig, error) {
	username := c.resolveUser()

	var authMethods []ssh.AuthMethod

	// Try ssh-agent first
	if agentAuth := c.agentAuth(); agentAuth != nil {
		authMethods = append(authMethods, agentAuth)
	}

	// Try key files from SSH config or defaults
	if keyAuth := c.keyAuth(); keyAuth != nil {
		authMethods = append(authMethods, keyAuth)
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication methods available (tried ssh-agent and key files)")
	}

	// Host key verification
	hostKeyCallback, err := c.hostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("host key verification: %w", err)
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}, nil
}

func (c *Conn) resolveUser() string {
	if c.target.User != "" {
		return c.target.User
	}
	if u := c.configGet("User"); u != "" {
		return u
	}
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "root"
}

func (c *Conn) resolveHost() string {
	if h := c.configGet("Hostname"); h != "" {
		return h
	}
	return c.target.Host
}

func (c *Conn) resolvePort() int {
	if c.target.Port != 0 {
		return c.target.Port
	}
	if p := c.configGet("Port"); p != "" {
		var port int
		fmt.Sscanf(p, "%d", &port)
		if port != 0 {
			return port
		}
	}
	return 22
}

func (c *Conn) agentAuth() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.DialTimeout("unix", sock, 2*time.Second)
	if err != nil {
		return nil
	}
	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

func (c *Conn) keyAuth() ssh.AuthMethod {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Check SSH config for IdentityFile
	var keyPaths []string
	if idFile := c.configGet("IdentityFile"); idFile != "" && idFile != "~/.ssh/identity" {
		// Expand ~ in path
		if len(idFile) > 0 && idFile[0] == '~' {
			idFile = filepath.Join(home, idFile[1:])
		}
		keyPaths = append(keyPaths, idFile)
	}

	// Default key locations
	keyPaths = append(keyPaths,
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	)

	var signers []ssh.Signer
	for _, path := range keyPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			// Could be passphrase-protected; skip for now
			// (ssh-agent handles most encrypted keys)
			continue
		}
		signers = append(signers, signer)
	}

	if len(signers) == 0 {
		return nil
	}
	return ssh.PublicKeys(signers...)
}

func (c *Conn) hostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// If known_hosts doesn't exist, create it
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0o700); err != nil {
			return nil, err
		}
		f, err := os.Create(knownHostsPath)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, err
	}

	// Wrap to auto-add unknown hosts (write to known_hosts)
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := cb(hostname, remote, key)
		if knownhosts.IsHostKeyChanged(err) {
			return fmt.Errorf("WARNING: remote host key has changed for %s. This could indicate a MITM attack", hostname)
		}
		if knownhosts.IsHostUnknown(err) {
			// Auto-add the key (TOFU - Trust On First Use)
			f, writeErr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0o644)
			if writeErr != nil {
				return fmt.Errorf("host key unknown and cannot write known_hosts: %w", writeErr)
			}
			defer f.Close()
			line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
			if _, writeErr := fmt.Fprintln(f, line); writeErr != nil {
				return fmt.Errorf("host key unknown and cannot write known_hosts: %w", writeErr)
			}
			return nil // Accepted
		}
		return err
	}, nil
}

func (c *Conn) keepaliveLoop() {
	ticker := time.NewTicker(c.keepalive)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			client := c.sshClient
			c.mu.RUnlock()

			if client == nil {
				return
			}

			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				// Connection dead, trigger reconnect
				go c.Reconnect()
				return
			}
		case <-c.done:
			return
		}
	}
}
