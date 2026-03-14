package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"golang.org/x/term"

	"github.com/Benjamin-Connelly/lookit/internal/config"
	"github.com/Benjamin-Connelly/lookit/internal/doctor"
	"github.com/Benjamin-Connelly/lookit/internal/export"
	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/manpages"
	"github.com/Benjamin-Connelly/lookit/internal/remote"
	"github.com/Benjamin-Connelly/lookit/internal/render"
	"github.com/Benjamin-Connelly/lookit/internal/tui"
	"github.com/Benjamin-Connelly/lookit/internal/web"
)

var version = "v0.4.0-dev"

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:     "lookit [path]",
	Short:   "Dual-mode markdown navigator with inter-document link navigation",
	Version: version,
	Long: `Lookit is a dual-mode markdown navigator (TUI + web) for browsing code,
markdown, and files. Features inter-document link navigation, backlinks,
broken link detection, fulltext search, and syntax highlighting for 50+
languages.

Usage:
  lookit                       Browse current directory
  lookit ~/docs                Browse a specific directory
  lookit README.md             Single-file mode (full-width preview)
  cat file.md | lookit         Render piped markdown

Remote browsing over SSH:
  lookit myhost:/path/to/docs  SCP-style remote path
  lookit user@host:/path       Explicit user
  lookit @docs                 Named remote from config

Configuration:
  ~/.config/lookit/config.yaml   Global config
  .lookit.toml / .lookit.yaml    Per-project config (auto-discovered)
  LOOKIT_* environment vars      Override any config key

TUI keybindings (press ? for full help):
  j/k, arrows   Navigate           /          Filter files / search preview
  Tab            Switch panels      Shift+Tab  Switch panels (reverse)
  Enter          Open file          f          Follow link
  h, Backspace   Go back            t          Table of contents
  b              Backlinks panel    i          Git info panel
  m              Bookmark / mark    M          Bookmarks panel
  V              Visual line select y          Copy permalink
  :              Command palette    Ctrl+G     Global heading jump
  Ctrl+T         Cycle theme        ?          Help overlay`,
	Example: `  lookit
  lookit ~/docs
  lookit README.md
  lookit devbox:/srv/docs
  echo "# Hello" | lookit
  lookit --keymap vim --theme dark ~/notes`,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: loadConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Detect piped stdin: render markdown and exit
		if stdinInfo, _ := os.Stdin.Stat(); stdinInfo != nil && stdinInfo.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			if len(data) == 0 {
				return nil
			}
			mdRenderer, err := render.NewMarkdownRenderer(cfg.Theme, 80)
			if err != nil {
				return fmt.Errorf("creating renderer: %w", err)
			}
			out, err := mdRenderer.Render(string(data))
			if err != nil {
				return fmt.Errorf("rendering: %w", err)
			}
			fmt.Print(out)
			return nil
		}

		// Check for remote path (host:/path syntax or --remote flag)
		if remoteHost, _ := cmd.Flags().GetString("remote"); remoteHost != "" {
			remotePath := "."
			if len(args) > 0 {
				remotePath = args[0]
			}
			remotePort, _ := cmd.Flags().GetInt("remote-port")
			target := &remote.Target{Host: remoteHost, Path: remotePath, Port: remotePort}
			return runRemote(target)
		}
		if len(args) > 0 {
			target := resolveRemoteTarget(args[0])
			if target != nil {
				return runRemote(target)
			}
		}

		root, initialFile, err := resolveRoot(args)
		if err != nil {
			return err
		}

		idx := index.New(root)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}

		// Build fulltext search index
		cacheDir, _ := os.UserCacheDir()
		if cacheDir != "" {
			cacheDir = filepath.Join(cacheDir, "lookit")
		}
		if err := idx.BuildFulltext(cacheDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: fulltext index unavailable: %v\n", err)
		}
		defer idx.CloseFulltext()

		links := index.NewLinkGraph()
		links.BuildFromIndex(idx)

		watcher, err := index.NewWatcher(idx, links, nil)
		if err != nil {
			return fmt.Errorf("starting watcher: %w", err)
		}
		defer watcher.Close()
		if err := watcher.Start(); err != nil {
			return fmt.Errorf("watching files: %w", err)
		}

		// Check minimum terminal size
		if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			if w < 80 || h < 24 {
				return fmt.Errorf("terminal too small (%dx%d). Lookit requires at least 80x24", w, h)
			}
		}

		model := tui.New(cfg, idx, links)
		if initialFile != "" {
			model.SelectFile(initialFile)
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve [path]",
	Short: "Start the web server",
	Long: `Start a web server for browsing markdown and code in a browser. Renders
markdown with GitHub Flavored Markdown, emoji, and syntax highlighting.
Watches for file changes and pushes live updates via Server-Sent Events.

API endpoints:
  /__api/files?q=term   Fuzzy file search (JSON)
  /__api/search?q=term  Full-text content search (JSON)
  /__api/graph          Link graph data (JSON)
  /graph                Interactive link graph visualization

The server adds security headers (CSP, X-Frame-Options, X-Content-Type-Options),
ETag caching, and skips auto-opening the browser when an SSH session is detected.`,
	Example: `  lookit serve
  lookit serve --port 3000 --open ~/docs
  lookit serve --css ./custom.css
  lookit serve --no-https`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _, err := resolveRoot(args)
		if err != nil {
			return err
		}
		port, _ := cmd.Flags().GetInt("port")
		if port != 0 {
			cfg.Server.Port = port
		}

		idx := index.New(root)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}

		// Build fulltext search index
		serveCacheDir, _ := os.UserCacheDir()
		if serveCacheDir != "" {
			serveCacheDir = filepath.Join(serveCacheDir, "lookit")
		}
		if err := idx.BuildFulltext(serveCacheDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: fulltext index unavailable: %v\n", err)
		}
		defer idx.CloseFulltext()

		links := index.NewLinkGraph()
		links.BuildFromIndex(idx)

		srv := web.New(cfg, idx, links)

		watcher, err := index.NewWatcher(idx, links, srv.OnFileChange)
		if err != nil {
			return fmt.Errorf("starting watcher: %w", err)
		}
		defer watcher.Close()
		if err := watcher.Start(); err != nil {
			return fmt.Errorf("watching files: %w", err)
		}

		err = srv.Start()
		// Suppress context deadline error on clean shutdown
		if err != nil && err.Error() == "context deadline exceeded" {
			return nil
		}
		return err
	},
}

var catCmd = &cobra.Command{
	Use:   "cat <file>",
	Short: "Render markdown or image to terminal",
	Long: `Render a markdown file or display an image directly in the terminal.

Markdown files are rendered with syntax highlighting and formatting.
Image files are displayed inline using your terminal's image protocol
(auto-detected: Kitty, iTerm2, WezTerm, Ghostty). Non-PNG images
(WebP, BMP, GIF, JPEG) are converted to PNG for protocol compatibility.

Supported image formats: PNG, JPG, WebP, BMP, GIF, SVG, ICO`,
	Example: `  lookit cat README.md
  lookit cat diagram.png
  lookit cat photo.webp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("file not found: %s", filePath)
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if isImageExt(ext) {
			protocol := render.DetectImageProtocol()
			out, err := render.RenderImageInline(filePath, protocol)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		}

		mdRenderer, err := render.NewMarkdownRenderer(cfg.Theme, 80)
		if err != nil {
			return fmt.Errorf("creating renderer: %w", err)
		}

		out, err := mdRenderer.RenderFile(filePath)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", filePath, err)
		}

		fmt.Print(out)
		return nil
	},
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg", ".ico":
		return true
	}
	return false
}

var exportCmd = &cobra.Command{
	Use:   "export [path]",
	Short: "Export markdown files to HTML or PDF",
	Long: `Export all markdown files to HTML with syntax highlighting. Processes files
in the target directory (respecting .gitignore), preserving the directory
structure under the output directory. Referenced images are copied alongside.

PDF export requires wkhtmltopdf to be installed. Default output directory
is "lookit-export" in the current directory.`,
	Example: `  lookit export
  lookit export ~/docs -f html -o ./site
  lookit export --format pdf --output ./pdfs`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _, err := resolveRoot(args)
		if err != nil {
			return err
		}

		formatStr, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = "lookit-export"
		}

		var format export.Format
		switch formatStr {
		case "html":
			format = export.FormatHTML
		case "pdf":
			format = export.FormatPDF
		default:
			return fmt.Errorf("unsupported format: %s", formatStr)
		}

		idx := index.New(root)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}

		opts := export.Options{
			Format:    format,
			OutputDir: output,
			Progress: func(current, total int, file string) {
				fmt.Printf("[%d/%d] %s\n", current, total, file)
			},
		}

		return export.Export(idx, opts)
	},
}

var graphCmd = &cobra.Command{
	Use:   "graph [path]",
	Short: "Output link graph in DOT format",
	Long: `Generate a Graphviz DOT representation of the inter-document link graph.
Nodes are markdown files, edges are links between them (standard markdown
links and [[wikilinks]]). Pipe to dot, neato, or other Graphviz tools to
render as an image.

An interactive graph is also available at /graph when using "lookit serve".`,
	Example: `  lookit graph
  lookit graph | dot -Tpng -o links.png
  lookit graph ~/docs | dot -Tsvg -o graph.svg`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _, err := resolveRoot(args)
		if err != nil {
			return err
		}

		idx := index.New(root)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}

		links := index.NewLinkGraph()
		links.BuildFromIndex(idx)

		fmt.Print(links.ToDOT())
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and diagnose issues",
	Long: `Run diagnostic checks on your environment and report results.

Checks: Go version, Git version, git repo detection, .gitignore presence,
terminal size and capabilities, config file loading, markdown file count,
large file warnings, and wkhtmltopdf availability (for PDF export).`,
	Example: `  lookit doctor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		checks := doctor.Run()
		doctor.Print(checks)
		return nil
	},
}

var genManCmd = &cobra.Command{
	Use:    "gen-man [output-dir]",
	Short:  "Generate man pages",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "./man"
		if len(args) > 0 {
			dir = args[0]
		}
		manDir := filepath.Join(dir, "man1")
		if err := os.MkdirAll(manDir, 0o755); err != nil {
			return err
		}
		header := &doc.GenManHeader{
			Title:   "LOOKIT",
			Section: "1",
			Source:  "lookit " + version,
		}
		if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
			return err
		}

		// Copy to embed directory so they're included in the binary
		embedDir := filepath.Join("internal", "manpages", "pages")
		if err := os.MkdirAll(embedDir, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(manDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(manDir, entry.Name()))
			if err != nil {
				continue
			}
			_ = os.WriteFile(filepath.Join(embedDir, entry.Name()), data, 0o644)
		}
		fmt.Printf("Generated %d man pages in %s and %s\n", len(entries), manDir, embedDir)
		return nil
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Set up shell completions for lookit",
	Long: `Set up shell completions so you get tab-completion for commands, flags, and file paths.

Run without arguments to auto-detect your shell and install interactively.
Run with a shell name to output the raw completion script (for custom setups).

Examples:
  lookit completion              # Interactive setup (recommended)
  lookit completion bash         # Print raw bash completion script
  lookit completion --install    # Auto-detect shell and install without prompts`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		install, _ := cmd.Flags().GetBool("install")

		// If a shell was specified with no --install, dump raw script (pipe-friendly)
		if len(args) == 1 && !install {
			return genCompletion(args[0], os.Stdout)
		}

		// Interactive / auto-install mode
		shell := detectShell()
		if len(args) == 1 {
			shell = args[0]
		}

		if shell == "" {
			fmt.Println("Could not detect your shell.")
			fmt.Println("Run with a shell name: lookit completion bash")
			return nil
		}

		dest, instruction := completionPath(shell)

		if install {
			return installCompletion(shell, dest, instruction)
		}

		// Interactive prompt
		fmt.Printf("Detected shell: %s\n\n", shell)
		fmt.Printf("This will install completions so you get tab-completion for:\n")
		fmt.Printf("  • Commands:  lookit <TAB>  →  cat, serve, export, doctor, ...\n")
		fmt.Printf("  • Flags:     lookit serve --<TAB>  →  --port, --open, ...\n")
		fmt.Printf("  • Files:     lookit cat <TAB>  →  file/directory completion\n\n")

		if dest != "" {
			fmt.Printf("Install to: %s\n", dest)
		} else {
			fmt.Printf("Setup: %s\n", instruction)
		}

		fmt.Printf("\nProceed? [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}

		return installCompletion(shell, dest, instruction)
	},
}

func detectShell() string {
	// Check SHELL env var
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := filepath.Base(shell)
		switch base {
		case "bash", "zsh", "fish":
			return base
		}
	}
	// Check $0 or PSModulePath for powershell
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}
	return ""
}

func completionPath(shell string) (dest, instruction string) {
	home, _ := os.UserHomeDir()
	switch shell {
	case "bash":
		// Prefer user-level bash-completion dir
		dir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
		return filepath.Join(dir, "lookit"), ""
	case "zsh":
		// Use ~/.zfunc if it exists, otherwise instruct to source
		dir := filepath.Join(home, ".zfunc")
		return filepath.Join(dir, "_lookit"), "Add to .zshrc: fpath=(~/.zfunc $fpath); autoload -Uz compinit && compinit"
	case "fish":
		dir := filepath.Join(home, ".config", "fish", "completions")
		return filepath.Join(dir, "lookit.fish"), ""
	case "powershell":
		return "", "Add to $PROFILE: lookit completion powershell | Out-String | Invoke-Expression"
	}
	return "", ""
}

func genCompletion(shell string, w *os.File) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(w)
	case "zsh":
		return rootCmd.GenZshCompletion(w)
	case "fish":
		return rootCmd.GenFishCompletion(w, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
	}
}

func installCompletion(shell, dest, instruction string) error {
	if dest == "" {
		// Can't auto-install (powershell) — show manual instructions
		fmt.Printf("\nAuto-install not supported for %s.\n", shell)
		fmt.Printf("Manual setup: %s\n", instruction)
		return nil
	}

	// Generate completion script to temp file, then copy to destination
	tmpFile, _ := os.CreateTemp("", "lookit-completion-*")
	defer os.Remove(tmpFile.Name())

	if err := genCompletion(shell, tmpFile); err != nil {
		tmpFile.Close()
		return fmt.Errorf("generating completion: %w", err)
	}
	tmpFile.Close()

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return err
	}
	// Ensure destination directory exists
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}

	fmt.Printf("\n✓ Completions installed to %s\n", dest)

	switch shell {
	case "bash":
		fmt.Println("\nTo activate now:  source " + dest)
		fmt.Println("It will load automatically in new terminals.")
	case "zsh":
		if instruction != "" {
			fmt.Printf("\nOne-time setup: %s\n", instruction)
		}
		fmt.Println("Then restart your shell or run: exec zsh")
	case "fish":
		fmt.Println("\nCompletions are active immediately in new Fish sessions.")
	}

	return nil
}

func init() {
	rootCmd.SetVersionTemplate("lookit {{.Version}}\n")
	rootCmd.Flags().BoolP("version", "V", false, "print version")

	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().String("theme", "", "color theme (light|dark|auto|ascii)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable verbose logging")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colors (ascii theme)")

	rootCmd.Flags().String("keymap", "", "keybinding preset (default|vim|emacs)")
	rootCmd.Flags().String("remote", "", "remote host (SSH config alias or user@host)")
	rootCmd.Flags().Int("remote-port", 0, "remote SSH port (default: from ssh config or 22)")

	serveCmd.Flags().IntP("port", "p", 0, "server port")
	serveCmd.Flags().Bool("open", false, "open browser after starting")
	serveCmd.Flags().Bool("no-https", false, "disable HTTPS even if certs exist")
	serveCmd.Flags().String("css", "", "path to custom CSS file")

	exportCmd.Flags().StringP("format", "f", "html", "export format (html|pdf)")
	exportCmd.Flags().StringP("output", "o", "", "output directory")

	completionCmd.Flags().Bool("install", false, "auto-detect shell and install without prompts")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(catCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(genManCmd)
}

func loadConfig(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")

	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// CLI flags override config file values
	if theme, _ := cmd.Flags().GetString("theme"); theme != "" {
		cfg.Theme = theme
	}
	if cmd.Flags().Lookup("keymap") != nil {
		if keymap, _ := cmd.Flags().GetString("keymap"); keymap != "" {
			cfg.Keymap = keymap
		}
	}

	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		cfg.Debug = true
	}
	if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
		cfg.Theme = "ascii"
	}

	// Merge serve-specific flags
	if cmd.Name() == "serve" || (cmd.Parent() != nil && cmd.Parent().Name() == "serve") {
		if noHTTPS, _ := cmd.Flags().GetBool("no-https"); noHTTPS {
			cfg.Server.NoHTTPS = true
		}
		if open, _ := cmd.Flags().GetBool("open"); open {
			cfg.Server.Open = true
		}
		if css, _ := cmd.Flags().GetString("css"); css != "" {
			cfg.Server.CustomCSS = css
		}
	}

	return cfg.Validate()
}

// resolveRoot returns the root directory and an optional initial file path.
// When the argument is a file, root is its parent directory and initialFile
// is the filename relative to root. When the argument is a directory,
// initialFile is empty.
func resolveRoot(args []string) (root string, initialFile string, err error) {
	rawRoot := cfg.Root
	if len(args) > 0 {
		rawRoot = args[0]
	}
	absRoot, err := filepath.Abs(rawRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolving root path: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", "", fmt.Errorf("root path %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		// Single file: use parent as root, file as initial selection
		return filepath.Dir(absRoot), filepath.Base(absRoot), nil
	}
	return absRoot, "", nil
}

// resolveRemoteTarget checks if the arg is a remote path spec or a named
// remote from config. Returns nil if the arg is a local path.
func resolveRemoteTarget(arg string) *remote.Target {
	// Try SCP-style parsing first (host:/path)
	if target := remote.ParseTarget(arg); target != nil {
		return target
	}

	// Check named remotes in config (e.g. @docs)
	if strings.HasPrefix(arg, "@") && cfg.Remotes != nil {
		name := arg[1:]
		if rc, ok := cfg.Remotes[name]; ok {
			return &remote.Target{
				Host: rc.Host,
				User: rc.User,
				Port: rc.Port,
				Path: rc.Path,
			}
		}
	}

	return nil
}

// runRemote handles the full lifecycle of browsing a remote host.
func runRemote(target *remote.Target) error {
	// Check minimum terminal size
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		if w < 80 || h < 24 {
			return fmt.Errorf("terminal too small (%dx%d). Lookit requires at least 80x24", w, h)
		}
	}

	fmt.Fprintf(os.Stderr, "Connecting to %s...\n", target.Display())

	// Establish SSH connection
	conn := remote.NewConn(*target)
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("connecting to %s: %w", target.Display(), err)
	}
	defer conn.Close()

	// Use the resolved target (~ expanded, relative paths resolved)
	resolved := conn.Target()

	fmt.Fprintf(os.Stderr, "Connected to %s. Indexing...\n", resolved.Display())

	// Use SFTP filesystem directly (no CacheOnReadFs — its directory
	// handling adds excessive round trips over SSH)
	sftpFs := remote.NewSFTPFs(conn.SFTP())

	// Resolve root: if target is a file, build a single-entry index
	// instead of walking the parent directory (which could be huge)
	root := resolved.Path
	var initialFile string
	var idx *index.Index

	info, err := sftpFs.Stat(root)
	if err != nil {
		return fmt.Errorf("stat remote path: %w", err)
	}

	if !info.IsDir() {
		// Single file: create index with just this entry
		initialFile = filepath.Base(root)
		root = filepath.Dir(root)
		idx = index.NewWithFs(root, sftpFs)
		idx.AddFile(resolved.Path, initialFile, info.Size(), info.ModTime())
	} else {
		// Directory: walk via SFTP
		idx = index.NewWithFs(root, sftpFs)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}
	}

	links := index.NewLinkGraph()

	fmt.Fprintf(os.Stderr, "Ready. Starting TUI...\n")

	// Create TUI with remote info (fulltext + link graph build in background)
	model := tui.New(cfg, idx, links)
	if initialFile != "" {
		model.SelectFile(initialFile)
	}
	model.SetRemoteInfo(&tui.RemoteInfo{
		Display: resolved.Display(),
		State:   conn.State().String(),
	})

	// Background: build fulltext + link graph, then poll for changes
	done := make(chan struct{})
	defer close(done)
	defer idx.CloseFulltext()
	lastRefresh := time.Now()
	go func() {
		// Build link graph and fulltext in background (reads files over SFTP)
		links.BuildFromIndex(idx)
		_ = idx.BuildFulltext("")

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(lastRefresh).Truncate(time.Second)
				model.SetRemoteInfo(&tui.RemoteInfo{
					Display:  resolved.Display(),
					State:    conn.State().String(),
					LastSync: fmt.Sprintf("refreshed %s ago", elapsed),
				})

				// Rebuild index every 15s to detect remote changes
				if time.Since(lastRefresh) >= 15*time.Second {
					if conn.State() == remote.ConnConnected {
						_ = idx.Rebuild()
						links.BuildFromIndex(idx)
						lastRefresh = time.Now()
					}
				}
			case <-done:
				return
			}
		}
	}()

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func main() {
	// Auto-install man pages on first run or version change
	manpages.Install(version)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
