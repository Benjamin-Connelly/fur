package tui

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestLoadingModelQuitsOnBuildDone drives the index-loading spinner and
// asserts it renders the building message and quits (capturing the build
// result) when the done channel reports (lookit-jiw).
func TestLoadingModelQuitsOnBuildDone(t *testing.T) {
	done := make(chan error, 1)
	sp := spinner.New()
	m := loadingModel{spinner: sp, root: "/some/tree", done: done}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Building index for /some/tree"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	done <- nil
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm, ok := tm.FinalModel(t).(loadingModel)
	if !ok {
		t.Fatalf("final model type %T", tm.FinalModel(t))
	}
	if !fm.built {
		t.Error("loader should have observed the build completion")
	}
	if fm.err != nil {
		t.Errorf("unexpected err: %v", fm.err)
	}
}

// TestLoadingModelCarriesBuildError confirms a build error propagates to the
// final model (and thus to ShowIndexLoading's return).
func TestLoadingModelCarriesBuildError(t *testing.T) {
	done := make(chan error, 1)
	sp := spinner.New()
	m := loadingModel{spinner: sp, root: "/x", done: done}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	wantErr := errors.New("walk failed")
	done <- wantErr
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm := tm.FinalModel(t).(loadingModel)
	if !errors.Is(fm.err, wantErr) {
		t.Errorf("err = %v, want %v", fm.err, wantErr)
	}
}
