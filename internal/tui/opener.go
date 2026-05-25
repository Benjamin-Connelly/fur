package tui

import (
	"os/exec"
	"strings"
)

// EditorCmd returns the exec.Cmd that openInEditor uses to spawn the user's
// configured editor on filePath. Exposed as a named function so the argv-safety
// invariant (Chain G) has a single chokepoint; production callers should use
// this helper rather than constructing exec.Command inline.
//
// Threat: filePath flows from the in-tree file selection. A hostile-directory
// adversary can plant files whose names begin with "-" (e.g. "-c", "-S",
// "-N"); when passed as argv[1] they are interpreted as flags by vim/ed/emacs
// rather than as positional file arguments, turning a filename into
// shell-command injection. See TestEditorCmd_FilenameFlagInjection_BUG.
//
// Mitigation: route filePath through safeFilenameArg, which prefixes "./" to
// any path beginning with "-" so the leading character is no longer a flag
// marker. "./" is preferred over a "--" separator because not every target
// tool honors "--" (xdg-open in particular doesn't).
func EditorCmd(editor, filePath string) *exec.Cmd {
	return exec.Command(editor, safeFilenameArg(filePath))
}

// OpenSystemCmd returns the exec.Cmd that openWithSystem uses to hand filePath
// to the platform's default application (xdg-open on Linux, open on macOS).
// Same argv-safety invariant and Chain G concern as EditorCmd — xdg-open in
// particular accepts flags like --help that a hostile filename can supply
// directly. See TestOpenSystemCmd_FilenameFlagInjection_BUG.
func OpenSystemCmd(opener, filePath string) *exec.Cmd {
	return exec.Command(opener, safeFilenameArg(filePath))
}

// safeFilenameArg neutralises a leading "-" in a filename so target programs
// (vim, emacs, ed, xdg-open) parse it as a positional file argument rather
// than as a flag. Absolute paths and paths whose first byte is anything other
// than "-" pass through unchanged.
func safeFilenameArg(filePath string) string {
	if strings.HasPrefix(filePath, "-") {
		return "./" + filePath
	}
	return filePath
}
