package remote

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// genSigner returns a fresh ed25519 ssh.Signer.
func genSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

// writeClientKey generates an ed25519 keypair, writes the private half to
// $HOME/.ssh/id_ed25519 (the path Conn.keyAuth reads), and returns the public
// key for the server to authorize.
func writeClientKey(t *testing.T, home string) ssh.PublicKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519"), pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("pub: %v", err)
	}
	return sshPub
}

// startSFTPServer starts an in-process SSH server on loopback that accepts the
// given client public key and serves SFTP. It returns the listen address and
// a stop function. No external network is used.
func startSFTPServer(t *testing.T, hostKey ssh.Signer, authorized ssh.PublicKey) (string, func()) {
	t.Helper()
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if string(key.Marshal()) == string(authorized.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unauthorized key")
		},
	}
	cfg.AddHostKey(hostKey)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for {
			nConn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go serveConn(nConn, cfg)
		}
	}()

	stop := func() {
		close(done)
		ln.Close()
	}
	return ln.Addr().String(), stop
}

func serveConn(nConn net.Conn, cfg *ssh.ServerConfig) {
	defer nConn.Close()
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "only sessions")
			continue
		}
		ch, chReqs, err := newChan.Accept()
		if err != nil {
			continue
		}
		go func(ch ssh.Channel, reqs <-chan *ssh.Request) {
			for req := range reqs {
				ok := req.Type == "subsystem" && len(req.Payload) >= 4 && string(req.Payload[4:]) == "sftp"
				if req.WantReply {
					_ = req.Reply(ok, nil)
				}
				if ok {
					srv, err := sftp.NewServer(ch)
					if err == nil {
						_ = srv.Serve()
						_ = srv.Close()
					}
					_ = ch.Close()
				}
			}
		}(ch, chReqs)
	}
}

// dialConn builds a Conn pointed at the in-process server with an isolated
// HOME (so known_hosts and the client key live in a tempdir), no agent.
func dialConn(t *testing.T, home, addr, remoteRoot string) *Conn {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("splithostport: %v", err)
	}
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return NewConn(Target{Host: host, Port: port, Path: remoteRoot})
}

// TestInProcessSSH_ConnectAndTOFU connects to an in-process SSH+SFTP server,
// asserts SFTP works, and that the host key is recorded in the isolated
// known_hosts (Trust On First Use).
func TestInProcessSSH_ConnectAndTOFU(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "") // force key-file auth, not agent

	clientPub := writeClientKey(t, home)
	hostKey := genSigner(t)
	addr, stop := startSFTPServer(t, hostKey, clientPub)
	defer stop()

	remoteRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteRoot, "doc.md"), []byte("# remote\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conn := dialConn(t, home, addr, remoteRoot)
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	if conn.State() != ConnConnected {
		t.Errorf("state = %v, want Connected", conn.State())
	}
	if conn.SFTP() == nil {
		t.Fatal("SFTP client nil")
	}
	if _, err := conn.SFTP().Stat(filepath.Join(remoteRoot, "doc.md")); err != nil {
		t.Errorf("SFTP Stat: %v", err)
	}

	// TOFU: the host key must now be pinned in the isolated known_hosts.
	kh, err := os.ReadFile(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if len(kh) == 0 {
		t.Error("known_hosts empty after first connect; TOFU did not pin the host key")
	}
}

// TestInProcessSSH_HostKeyChangeRejected pins a host key via TOFU, then
// restarts the server with a DIFFERENT host key on the same address and
// asserts the connection is refused (MITM defense).
func TestInProcessSSH_HostKeyChangeRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	clientPub := writeClientKey(t, home)
	remoteRoot := t.TempDir()

	// First server + connect to pin the key.
	hostKey1 := genSigner(t)
	addr, stop1 := startSFTPServer(t, hostKey1, clientPub)
	conn1 := dialConn(t, home, addr, remoteRoot)
	if err := conn1.Connect(); err != nil {
		stop1()
		t.Fatalf("first Connect: %v", err)
	}
	conn1.Close()
	stop1()

	// Reuse the SAME host:port with a DIFFERENT host key.
	host, portStr, _ := net.SplitHostPort(addr)
	ln, err := reuseListen(host, portStr)
	if err != nil {
		t.Skipf("could not rebind %s (port reuse race): %v", addr, err)
	}
	hostKey2 := genSigner(t)
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	cfg.AddHostKey(hostKey2)
	done := make(chan struct{})
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go serveConn(nc, cfg)
		}
	}()
	defer func() { close(done); ln.Close() }()

	conn2 := dialConn(t, home, addr, remoteRoot)
	err = conn2.Connect()
	if err == nil {
		conn2.Close()
		t.Fatal("Connect succeeded after host key changed; MITM not detected")
	}
}

// reuseListen rebinds a specific host:port with SO_REUSEADDR-ish best effort.
func reuseListen(host, port string) (net.Listener, error) {
	// Retry briefly: the previous listener may not have fully released yet.
	var lastErr error
	for i := 0; i < 20; i++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
		if err == nil {
			return ln, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

// TestInProcessSSH_WrongKeyRejected asserts authentication fails when the
// client key is not authorized by the server.
func TestInProcessSSH_WrongKeyRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	_ = writeClientKey(t, home) // client uses this key...

	// ...but the server authorizes a DIFFERENT key.
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	authorized, _ := ssh.NewPublicKey(otherPub)
	hostKey := genSigner(t)
	addr, stop := startSFTPServer(t, hostKey, authorized)
	defer stop()

	conn := dialConn(t, home, addr, t.TempDir())
	if err := conn.Connect(); err == nil {
		conn.Close()
		t.Fatal("Connect succeeded with an unauthorized client key")
	}
}
