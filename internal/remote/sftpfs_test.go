package remote

import (
	"os"
	"testing"

	"github.com/spf13/afero"
)

func TestSFTPFsName(t *testing.T) {
	fs := &SFTPFs{}
	if fs.Name() != "SFTPFs" {
		t.Errorf("Name() = %q, want %q", fs.Name(), "SFTPFs")
	}
}

func TestSFTPFsWriteOpsReturnReadOnly(t *testing.T) {
	fs := &SFTPFs{}

	if _, err := fs.Create("test"); err != errReadOnly {
		t.Errorf("Create() = %v, want errReadOnly", err)
	}
	if err := fs.Mkdir("test", 0o755); err != errReadOnly {
		t.Errorf("Mkdir() = %v, want errReadOnly", err)
	}
	if err := fs.MkdirAll("test", 0o755); err != errReadOnly {
		t.Errorf("MkdirAll() = %v, want errReadOnly", err)
	}
	if err := fs.Remove("test"); err != errReadOnly {
		t.Errorf("Remove() = %v, want errReadOnly", err)
	}
	if err := fs.RemoveAll("test"); err != errReadOnly {
		t.Errorf("RemoveAll() = %v, want errReadOnly", err)
	}
	if err := fs.Rename("a", "b"); err != errReadOnly {
		t.Errorf("Rename() = %v, want errReadOnly", err)
	}
	if err := fs.Chmod("test", 0o644); err != errReadOnly {
		t.Errorf("Chmod() = %v, want errReadOnly", err)
	}
	if err := fs.Chown("test", 0, 0); err != errReadOnly {
		t.Errorf("Chown() = %v, want errReadOnly", err)
	}
}

func TestSFTPFsOpenFileRejectsWriteFlags(t *testing.T) {
	fs := &SFTPFs{}

	flags := []int{
		os.O_WRONLY,
		os.O_RDWR,
		os.O_CREATE,
		os.O_TRUNC,
		os.O_APPEND,
	}

	for _, flag := range flags {
		if _, err := fs.OpenFile("test", flag, 0); err != errReadOnly {
			t.Errorf("OpenFile(flag=%d) = %v, want errReadOnly", flag, err)
		}
	}
}

func TestSftpFileWriteOps(t *testing.T) {
	f := &sftpFile{}

	if _, err := f.Write([]byte("x")); err != errReadOnly {
		t.Errorf("Write() = %v, want errReadOnly", err)
	}
	if _, err := f.WriteAt([]byte("x"), 0); err != errReadOnly {
		t.Errorf("WriteAt() = %v, want errReadOnly", err)
	}
	if _, err := f.WriteString("x"); err != errReadOnly {
		t.Errorf("WriteString() = %v, want errReadOnly", err)
	}
	if err := f.Truncate(0); err != errReadOnly {
		t.Errorf("Truncate() = %v, want errReadOnly", err)
	}
	if err := f.Sync(); err != nil {
		t.Errorf("Sync() = %v, want nil", err)
	}
	if _, err := f.Readdir(0); err == nil {
		t.Error("Readdir() on file should error")
	}
	if _, err := f.Readdirnames(0); err == nil {
		t.Error("Readdirnames() on file should error")
	}
}

func TestSftpDirOps(t *testing.T) {
	d := &sftpDir{path: "/test"}

	if d.Name() != "/test" {
		t.Errorf("Name() = %q, want %q", d.Name(), "/test")
	}
	if err := d.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	if _, err := d.Read(make([]byte, 1)); err == nil {
		t.Error("Read() on dir should error")
	}
	if _, err := d.ReadAt(make([]byte, 1), 0); err == nil {
		t.Error("ReadAt() on dir should error")
	}
	if _, err := d.Seek(0, 0); err == nil {
		t.Error("Seek() on dir should error")
	}
	if _, err := d.Write([]byte("x")); err != errReadOnly {
		t.Errorf("Write() = %v, want errReadOnly", err)
	}
	if _, err := d.WriteAt([]byte("x"), 0); err != errReadOnly {
		t.Errorf("WriteAt() = %v, want errReadOnly", err)
	}
	if _, err := d.WriteString("x"); err != errReadOnly {
		t.Errorf("WriteString() = %v, want errReadOnly", err)
	}
	if err := d.Truncate(0); err != errReadOnly {
		t.Errorf("Truncate() = %v, want errReadOnly", err)
	}
}

func TestNewSFTPFsImplementsAferoFs(t *testing.T) {
	// Compile-time check that SFTPFs implements afero.Fs
	var _ afero.Fs = (*SFTPFs)(nil)
}
