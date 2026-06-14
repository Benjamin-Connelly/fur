package remote

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// TestConnAgentAuth exercises the ssh-agent auth path: an in-process agent
// (unix socket) holds the client key, SSH_AUTH_SOCK points at it, and Connect
// authenticates through it (lookit-9py.2.1 coverage for conn.agentAuth + the
// agentConn close path).
func TestConnAgentAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatalf("agent add: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	sock := filepath.Join(home, "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("agent listen: %v", err)
	}
	agentDone := make(chan struct{})
	go func() {
		defer close(agentDone)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(keyring, c)
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", sock)

	hostKey := genSigner(t)
	addr, stop := startSFTPServer(t, hostKey, sshPub)
	defer stop()

	conn := dialConn(t, home, addr, t.TempDir())
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect via agent: %v", err)
	}
	if conn.State() != ConnConnected {
		t.Errorf("state = %v, want Connected", conn.State())
	}
	conn.Close() // closes the agent connection (covers that branch)

	ln.Close()
	<-agentDone
}

// TestConnKeepaliveTick drives the keepalive loop's tick branch by using a
// short interval and waiting for at least one keepalive round-trip.
func TestConnKeepaliveTick(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	clientPub := writeClientKey(t, home)
	hostKey := genSigner(t)
	addr, stop := startSFTPServer(t, hostKey, clientPub)
	defer stop()

	conn := dialConn(t, home, addr, t.TempDir())
	conn.keepalive = 30 * time.Millisecond // short, so a tick fires during the test
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(120 * time.Millisecond) // allow a few keepalive ticks
	if conn.State() != ConnConnected {
		t.Errorf("keepalive should keep the connection alive, state = %v", conn.State())
	}
}
