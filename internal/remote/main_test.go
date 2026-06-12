package remote

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs goleak after the remote suite. Conn spawns a keepalive loop
// and holds an ssh-agent socket connection; the in-process SSH/SFTP test
// servers spawn accept/serve goroutines. A connection left unclosed (missing
// Conn.Close, or a server whose listener wasn't stopped) surfaces here
// (lookit-9py.2.7).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
