package index

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs goleak after the index suite. The fsnotify watcher and Bleve
// fulltext index both spawn goroutines; a watcher left unclosed or a fulltext
// index left open would surface here (lookit-9py.2.7).
//
// bleve's analysis-queue workers are process-global library goroutines with
// no per-call shutdown and are ignored; regexp2's clock is pulled in
// transitively the same way.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2.runClock"),
		goleak.IgnoreAnyFunction("github.com/blevesearch/bleve_index_api.AnalysisWorker"),
	)
}
