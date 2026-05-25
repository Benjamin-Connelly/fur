package tui

import "os/exec"

// EditorCmd returns the exec.Cmd that openInEditor uses to spawn the user's
// configured editor on filePath. Exposed as a named function so the argv-safety
// invariant (Chain G) can be asserted by tests; production callers should use
// this helper rather than constructing exec.Command inline so that any future
// fix to argv handling applies uniformly.
//
// Threat: filePath flows from the in-tree file selection. A hostile-directory
// adversary can plant files whose names begin with "-" (e.g. "-c", "-S",
// "-N"); when passed as argv[1] they are interpreted as flags by vim/ed/emacs
// rather than as positional file arguments, turning a filename into
// shell-command injection.
//
// Mitigation (deferred to lookit-9py.3.11.2 Chain G Fix): prepend "--" before
// the filename, or prefix relative paths with "./". See
// TestEditorCmd_FilenameFlagInjection_BUG.
func EditorCmd(editor, filePath string) *exec.Cmd {
	return exec.Command(editor, filePath)
}

// OpenSystemCmd returns the exec.Cmd that openWithSystem uses to hand filePath
// to the platform's default application (xdg-open on Linux, open on macOS).
// Same argv-safety invariant and Chain G concern as EditorCmd — xdg-open in
// particular accepts flags like --help that a hostile filename can supply
// directly.
//
// See TestOpenSystemCmd_FilenameFlagInjection_BUG.
func OpenSystemCmd(opener, filePath string) *exec.Cmd {
	return exec.Command(opener, filePath)
}
