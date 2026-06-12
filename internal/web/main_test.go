package web

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs goleak after the web suite to catch goroutine leaks — the SSE
// broker spawns a goroutine per client plus a broadcast loop, so a leaked
// client goroutine (e.g. a slow-client backpressure bug or a missing Stop)
// surfaces here (lookit-9py.2.7).
//
// The ignored functions are process-global background goroutines started by
// third-party libraries with no per-instance shutdown hook: regexp2's fast
// clock (pulled in by chroma highlighting) and bleve's analysis-queue workers.
// They are not fur-owned leaks.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2.runClock"),
		goleak.IgnoreAnyFunction("github.com/blevesearch/bleve_index_api.AnalysisWorker"),
	)
}
