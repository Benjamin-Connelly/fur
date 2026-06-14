package remote

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// connectInProcess spins the in-process SSH+SFTP server, connects a Conn to it
// (path = remoteRoot), and returns the live connection. Shared by the SFTPFs
// and reconnect tests.
func connectInProcess(t *testing.T, remoteRoot string) *Conn {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	clientPub := writeClientKey(t, home)
	hostKey := genSigner(t)
	addr, stop := startSFTPServer(t, hostKey, clientPub)
	t.Cleanup(stop)

	conn := dialConn(t, home, addr, remoteRoot)
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestSFTPFsReadPaths exercises the read side of the afero.Fs-over-SFTP
// (Open/Read, Stat, Readdir/Readdirnames, Walk) against the in-process SFTP
// server (lookit-9py.2.1 coverage for internal/remote/sftpfs.go).
func TestSFTPFsReadPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.md"), []byte("# B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conn := connectInProcess(t, root)
	fs := NewSFTPFs(conn.SFTP())

	// Open + Read a file.
	f, err := fs.Open(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "# A\nhello\n" {
		t.Errorf("read content = %q", data)
	}

	// Stat.
	fi, err := fs.Stat(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.IsDir() || fi.Size() == 0 {
		t.Errorf("Stat: unexpected %+v", fi)
	}

	// Readdir / Readdirnames on the root directory.
	d, err := fs.Open(root)
	if err != nil {
		t.Fatalf("Open dir: %v", err)
	}
	defer d.Close()
	infos, err := d.Readdir(-1)
	if err != nil {
		t.Fatalf("Readdir: %v", err)
	}
	if len(infos) != 2 { // a.md + sub
		t.Errorf("Readdir count = %d, want 2", len(infos))
	}

	d2, _ := fs.Open(root)
	defer d2.Close()
	names, err := d2.Readdirnames(-1)
	if err != nil {
		t.Fatalf("Readdirnames: %v", err)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "a.md" {
		t.Errorf("Readdirnames = %v", names)
	}

	// Walk the tree and confirm it reaches the nested file.
	var walked []string
	if err := fs.(*SFTPFs).Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			walked = append(walked, filepath.Base(path))
		}
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}
	sort.Strings(walked)
	if len(walked) != 2 || walked[0] != "a.md" || walked[1] != "b.md" {
		t.Errorf("Walk visited %v, want [a.md b.md]", walked)
	}
}

// TestSFTPFsWriteOpsRejected confirms every mutating op returns the read-only
// error against a live connection.
func TestSFTPFsWriteOpsRejected(t *testing.T) {
	root := t.TempDir()
	conn := connectInProcess(t, root)
	fs := NewSFTPFs(conn.SFTP())

	if _, err := fs.Create("x"); err == nil {
		t.Error("Create should be read-only")
	}
	if err := fs.Remove("x"); err == nil {
		t.Error("Remove should be read-only")
	}
	if err := fs.Mkdir("x", 0o755); err == nil {
		t.Error("Mkdir should be read-only")
	}
	if err := fs.Chmod("x", 0o644); err == nil {
		t.Error("Chmod should be read-only")
	}
}

// TestSFTPFileMethods covers the afero.File surface of an opened remote file:
// the no-op/read-only overrides and the not-a-directory readdir paths.
func TestSFTPFileMethods(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	conn := connectInProcess(t, root)
	fs := NewSFTPFs(conn.SFTP())

	f, err := fs.Open(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	if err := f.Sync(); err != nil {
		t.Errorf("Sync = %v, want nil", err)
	}
	if err := f.Truncate(0); err == nil {
		t.Error("Truncate should be read-only")
	}
	if _, err := f.WriteString("x"); err == nil {
		t.Error("WriteString should be read-only")
	}
	if _, err := f.Write([]byte("x")); err == nil {
		t.Error("Write should be read-only")
	}
	if _, err := f.WriteAt([]byte("x"), 0); err == nil {
		t.Error("WriteAt should be read-only")
	}
	if _, err := f.Readdir(-1); err == nil {
		t.Error("Readdir on a file should error")
	}
	if _, err := f.Readdirnames(-1); err == nil {
		t.Error("Readdirnames on a file should error")
	}

	// Fs-level mutators that weren't covered by the rejected-write test.
	if err := fs.Chtimes("a.md", time.Now(), time.Now()); err == nil {
		t.Error("Chtimes should be read-only")
	}
	if err := fs.Chown("a.md", 0, 0); err == nil {
		t.Error("Chown should be read-only")
	}
	if err := fs.Rename("a.md", "b.md"); err == nil {
		t.Error("Rename should be read-only")
	}
	if err := fs.RemoveAll("a.md"); err == nil {
		t.Error("RemoveAll should be read-only")
	}
	if err := fs.MkdirAll("d", 0o755); err == nil {
		t.Error("MkdirAll should be read-only")
	}
}

// TestConnResolveRemotePath exercises resolveRemotePath's home-expansion
// branches ("", ".", "~", "~/sub") against the in-process server.
func TestConnResolveRemotePath(t *testing.T) {
	for _, path := range []string{".", "~", "~/sub"} {
		t.Run(path, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("SSH_AUTH_SOCK", "")
			clientPub := writeClientKey(t, home)
			hostKey := genSigner(t)
			addr, stop := startSFTPServer(t, hostKey, clientPub)
			t.Cleanup(stop)

			conn := dialConn(t, home, addr, path)
			if err := conn.Connect(); err != nil {
				t.Fatalf("Connect(path=%q): %v", path, err)
			}
			t.Cleanup(func() { conn.Close() })
			// After resolution the path must be absolute (home expanded).
			if got := conn.target.Path; got == "" || got[0] != '/' {
				t.Errorf("resolved path = %q, want absolute", got)
			}
		})
	}
}

// TestConnReconnect drives Conn.Reconnect against the in-process server.
func TestConnReconnect(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	conn := connectInProcess(t, root)

	if err := conn.Reconnect(); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if conn.State() != ConnConnected {
		t.Errorf("state after reconnect = %v, want Connected", conn.State())
	}
	// SFTP should work again post-reconnect.
	if _, err := conn.SFTP().Stat(filepath.Join(root, "f.md")); err != nil {
		t.Errorf("SFTP after reconnect: %v", err)
	}
}
