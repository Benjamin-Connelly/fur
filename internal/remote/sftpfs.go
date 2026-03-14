package remote

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/afero"
)

var errReadOnly = errors.New("read-only filesystem")

// SFTPFs implements afero.Fs backed by an SFTP client.
// Only read operations are supported; write operations return errReadOnly.
type SFTPFs struct {
	client *sftp.Client
}

// NewSFTPFs creates a read-only afero.Fs backed by an SFTP connection.
func NewSFTPFs(client *sftp.Client) afero.Fs {
	return &SFTPFs{client: client}
}

func (s *SFTPFs) Name() string { return "SFTPFs" }

func (s *SFTPFs) Open(name string) (afero.File, error) {
	return s.OpenFile(name, os.O_RDONLY, 0)
}

func (s *SFTPFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 {
		return nil, errReadOnly
	}

	info, err := s.client.Stat(name)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return &sftpDir{client: s.client, path: name}, nil
	}

	f, err := s.client.Open(name)
	if err != nil {
		return nil, err
	}
	return &sftpFile{File: f}, nil
}

func (s *SFTPFs) Stat(name string) (os.FileInfo, error) {
	return s.client.Stat(name)
}

func (s *SFTPFs) Create(name string) (afero.File, error)            { return nil, errReadOnly }
func (s *SFTPFs) Mkdir(name string, perm os.FileMode) error         { return errReadOnly }
func (s *SFTPFs) MkdirAll(path string, perm os.FileMode) error      { return errReadOnly }
func (s *SFTPFs) Remove(name string) error                          { return errReadOnly }
func (s *SFTPFs) RemoveAll(path string) error                       { return errReadOnly }
func (s *SFTPFs) Rename(oldname, newname string) error              { return errReadOnly }
func (s *SFTPFs) Chmod(name string, mode os.FileMode) error         { return errReadOnly }
func (s *SFTPFs) Chown(name string, uid, gid int) error             { return errReadOnly }
func (s *SFTPFs) Chtimes(name string, atime, mtime time.Time) error { return errReadOnly }

// sftpFile wraps sftp.File to satisfy afero.File.
type sftpFile struct {
	*sftp.File
}

func (f *sftpFile) Readdir(count int) ([]os.FileInfo, error)  { return nil, errors.New("not a directory") }
func (f *sftpFile) Readdirnames(count int) ([]string, error)  { return nil, errors.New("not a directory") }
func (f *sftpFile) Sync() error                               { return nil }
func (f *sftpFile) Truncate(size int64) error                 { return errReadOnly }
func (f *sftpFile) WriteString(s string) (ret int, err error) { return 0, errReadOnly }
func (f *sftpFile) Write(p []byte) (int, error)               { return 0, errReadOnly }
func (f *sftpFile) WriteAt(p []byte, off int64) (int, error)  { return 0, errReadOnly }

// sftpDir represents a remote directory as an afero.File.
type sftpDir struct {
	client  *sftp.Client
	path    string
	entries []os.FileInfo
	offset  int
}

func (d *sftpDir) Close() error { return nil }
func (d *sftpDir) Name() string { return d.path }

func (d *sftpDir) Stat() (os.FileInfo, error) {
	return d.client.Stat(d.path)
}

func (d *sftpDir) Readdir(count int) ([]os.FileInfo, error) {
	if d.entries == nil {
		entries, err := d.client.ReadDir(d.path)
		if err != nil {
			return nil, err
		}
		d.entries = entries
	}

	if count <= 0 {
		result := d.entries[d.offset:]
		d.offset = len(d.entries)
		if len(result) == 0 {
			return result, io.EOF
		}
		return result, nil
	}

	end := d.offset + count
	if end > len(d.entries) {
		end = len(d.entries)
	}
	result := d.entries[d.offset:end]
	d.offset = end

	if d.offset >= len(d.entries) {
		return result, io.EOF
	}
	return result, nil
}

func (d *sftpDir) Readdirnames(count int) ([]string, error) {
	entries, err := d.Readdir(count)
	if err != nil && err != io.EOF {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names, err
}

func (d *sftpDir) Read(p []byte) (int, error)                   { return 0, errors.New("is a directory") }
func (d *sftpDir) ReadAt(p []byte, off int64) (int, error)      { return 0, errors.New("is a directory") }
func (d *sftpDir) Seek(offset int64, whence int) (int64, error) { return 0, errors.New("is a directory") }
func (d *sftpDir) Write(p []byte) (int, error)                  { return 0, errReadOnly }
func (d *sftpDir) WriteAt(p []byte, off int64) (int, error)     { return 0, errReadOnly }
func (d *sftpDir) WriteString(s string) (ret int, err error)    { return 0, errReadOnly }
func (d *sftpDir) Sync() error                                  { return nil }
func (d *sftpDir) Truncate(size int64) error                    { return errReadOnly }
