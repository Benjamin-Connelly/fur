package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Benjamin-Connelly/lookit/internal/config"
	"github.com/Benjamin-Connelly/lookit/internal/doctor"
	"github.com/Benjamin-Connelly/lookit/internal/export"
	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/render"
	"github.com/Benjamin-Connelly/lookit/internal/tui"
	"github.com/Benjamin-Connelly/lookit/internal/web"
)

var version = "v0.1.0"

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "lookit [path]",
	Short: "Dual-mode markdown navigator with inter-document link navigation",
	Long: `Lookit is a dual-mode markdown navigator that provides both TUI and web
interfaces for browsing code, markdown, and files. Features inter-document
link navigation with history, backlinks, and broken link detection.`,
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

		root, err := resolveRoot(args)
		if err != nil {
			return err
		}

		idx := index.New(root)
		if err := idx.Build(); err != nil {
			return fmt.Errorf("building index: %w", err)
		}

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

		model := tui.New(cfg, idx, links)
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve [path]",
	Short: "Start the web server",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := resolveRoot(args)
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
	Short: "Render markdown to terminal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("file not found: %s", filePath)
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

var exportCmd = &cobra.Command{
	Use:   "export [path]",
	Short: "Export markdown files to HTML or PDF",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := resolveRoot(args)
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

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and diagnose issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		checks := doctor.Run()
		doctor.Print(checks)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("lookit %s\n", version)
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
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().String("theme", "", "color theme (light|dark|auto)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable verbose logging")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colors (ascii theme)")

	rootCmd.Flags().String("keymap", "", "keybinding preset (default|vim|emacs)")

	serveCmd.Flags().IntP("port", "p", 0, "server port")
	serveCmd.Flags().Bool("open", false, "open browser after starting")
	serveCmd.Flags().Bool("no-https", false, "disable HTTPS even if certs exist")

	exportCmd.Flags().StringP("format", "f", "html", "export format (html|pdf)")
	exportCmd.Flags().StringP("output", "o", "", "output directory")

	completionCmd.Flags().Bool("install", false, "auto-detect shell and install without prompts")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(catCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
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
	}

	return cfg.Validate()
}

func resolveRoot(args []string) (string, error) {
	root := cfg.Root
	if len(args) > 0 {
		root = args[0]
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving root path: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("root path %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root path %q is not a directory", absRoot)
	}
	return absRoot, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
