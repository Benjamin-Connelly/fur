package tui

import (
	"strings"
	"testing"
)

// TestEditorCmd_FilenameFlagInjection_BUG is a Chain G proof-of-concept.
//
// EditorCmd is called from openInEditor with the selected file's relative
// path. A hostile-directory adversary (per the audit threat model) can plant
// files whose names begin with "-" — e.g. "-c", "-S /tmp/evil.vim", "-N".
// When passed as argv[1] to vim/ed/emacs, these names are parsed as flags
// instead of as positional file arguments:
//
//   - vim's -c <cmd> / -S <script> let an attacker execute ex-commands
//     (including :!shell) at startup.
//   - emacs's --eval / -f / -l likewise execute attacker-supplied Lisp.
//   - ed's -p / -s / -l are at minimum a behavior-change primitive.
//
// The fix (lookit-9py.3.11.2) is to either prepend "--" before the filename
// (so the editor treats subsequent argv as positional) or to prefix relative
// paths with "./" (so leading-dash names lose their flag shape). This test
// asserts one of those mitigations is present in the argv.
//
// References: lookit-9py.3.11.1; bd memory
// "argv-safe-exec-only-exec-command-name-args"; docs/audit-prompt.md
// (Chain G — Filename → exec injection via $EDITOR/xdg-open).
func TestEditorCmd_FilenameFlagInjection_BUG(t *testing.T) {
	hostile := []string{
		"-c:!touch /tmp/pwned",
		"-S /tmp/evil.vim",
		"-N",
		"--cmd=:!evil",
	}

	for _, filename := range hostile {
		t.Run(filename, func(t *testing.T) {
			cmd := EditorCmd("vim", filename)
			if !argvFilenameIsSafe(cmd.Args, filename) {
				t.Errorf("EditorCmd(%q, %q) produced unsafe argv=%v: "+
					"filename appears in a flag-shaped position. Mitigation: "+
					"insert \"--\" before the filename, or prefix relative "+
					"paths with \"./\".",
					"vim", filename, cmd.Args)
			}
		})
	}
}

// TestOpenSystemCmd_FilenameFlagInjection_BUG is the xdg-open / open variant
// of Chain G. Same threat shape: a hostile filename starting with "-" is
// passed unguarded to the platform's default-application launcher.
//
// xdg-open in particular treats argv with leading "-" as its own flags
// (--help, --version, etc.); behavior with arbitrary "-foo" depends on the
// version but is at minimum unpredictable and at worst a vector for
// downstream tools the launcher invokes.
//
// Same mitigations as EditorCmd apply.
func TestOpenSystemCmd_FilenameFlagInjection_BUG(t *testing.T) {
	hostile := []string{
		"-help",
		"--version",
		"-x.txt",
	}

	for _, filename := range hostile {
		t.Run(filename, func(t *testing.T) {
			cmd := OpenSystemCmd("xdg-open", filename)
			if !argvFilenameIsSafe(cmd.Args, filename) {
				t.Errorf("OpenSystemCmd(%q, %q) produced unsafe argv=%v: "+
					"filename appears in a flag-shaped position. Mitigation: "+
					"insert \"--\" before the filename, or prefix relative "+
					"paths with \"./\".",
					"xdg-open", filename, cmd.Args)
			}
		})
	}
}

// argvFilenameIsSafe reports whether the supplied argv passes filename to a
// child process in a way that cannot be misinterpreted as a flag.
//
// Two recognised mitigations:
//  1. Argv contains "--" before filename (everything after "--" is positional).
//  2. The argv entry corresponding to filename is prefixed with "./" so the
//     leading character is no longer "-".
//
// Returns true on safe argv; false if the bare hostile filename appears in a
// flag-shaped position.
func argvFilenameIsSafe(args []string, filename string) bool {
	seenSeparator := false
	for i := 1; i < len(args); i++ { // args[0] is the program name
		a := args[i]
		if a == "--" {
			seenSeparator = true
			continue
		}
		if a == filename {
			// Bare hostile filename in flag-shaped position is unsafe unless
			// the "--" separator already appeared.
			if seenSeparator {
				return true
			}
			return false
		}
		if strings.HasSuffix(a, filename) && strings.HasPrefix(a, "./") {
			// "./" prefix neutralises the leading "-".
			return true
		}
	}
	// filename not found in argv at all → vacuously safe (some other handling).
	return true
}
