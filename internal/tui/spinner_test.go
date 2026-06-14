package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
)

// TestRemoteSpinnerLifecycle is the regression guard for lookit-rl8: a
// RemoteInfoMsg in a transient state starts the spinner tick loop; the loop
// keeps ticking while transient and stops once the connection settles. State
// updates flow through the message (not direct model mutation), so the render
// loop never races the poll goroutine.
func TestRemoteSpinnerLifecycle(t *testing.T) {
	m := testModel(t)

	// Entering "Reconnecting" starts the spinner.
	updated, cmd := m.Update(RemoteInfoMsg{Info: RemoteInfo{Display: "u@h:/p", State: "Reconnecting"}})
	m = updated.(*Model)
	if m.remoteInfo == nil || m.remoteInfo.State != "Reconnecting" {
		t.Fatalf("remoteInfo not updated from message: %+v", m.remoteInfo)
	}
	if !m.spinnerOn {
		t.Error("spinner should be running while reconnecting")
	}
	if cmd == nil {
		t.Error("expected a spinner tick command on entering reconnecting state")
	}

	// A tick while still reconnecting keeps the loop going.
	updated, cmd = m.Update(spinner.TickMsg{})
	m = updated.(*Model)
	if cmd == nil {
		t.Error("spinner should keep ticking while reconnecting")
	}

	// Once connected, the next tick stops the loop.
	updated, _ = m.Update(RemoteInfoMsg{Info: RemoteInfo{Display: "u@h:/p", State: "Connected"}})
	m = updated.(*Model)
	updated, cmd = m.Update(spinner.TickMsg{})
	m = updated.(*Model)
	if m.spinnerOn {
		t.Error("spinner should stop once connected")
	}
	if cmd != nil {
		t.Error("no further tick expected once connected")
	}
}

// TestRemoteSpinnerNotStartedWhenConnected confirms a steady-state update does
// not spin up the tick loop.
func TestRemoteSpinnerNotStartedWhenConnected(t *testing.T) {
	m := testModel(t)
	updated, cmd := m.Update(RemoteInfoMsg{Info: RemoteInfo{Display: "u@h:/p", State: "Connected"}})
	m = updated.(*Model)
	if m.spinnerOn {
		t.Error("spinner should not run in the Connected state")
	}
	if cmd != nil {
		t.Error("no spinner tick expected in the Connected state")
	}
}
