package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Benjamin-Connelly/lookit/internal/config"
	"github.com/Benjamin-Connelly/lookit/internal/doctor"
	"github.com/Benjamin-Connelly/lookit/internal/export"
	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/spf13/cobra"
)

const version = "v0.1.0"

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
		root, err := resolveRoot(args)
		if err != nil {
			return err
		}
		fmt.Printf("launching TUI at %s\n", root)
		return nil
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
		if port == 0 {
			port = cfg.Server.Port
		}
		fmt.Printf("starting web server on :%d serving %s\n", port, root)
		return nil
	},
}

var catCmd = &cobra.Command{
	Use:   "cat <file>",
	Short: "Render markdown to terminal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("rendering %s\n", args[0])
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
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for lookit.

To load completions:

Bash:
  $ source <(lookit completion bash)

Zsh:
  $ source <(lookit completion zsh)

Fish:
  $ lookit completion fish | source
`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().String("theme", "", "color theme (light|dark|auto)")

	rootCmd.Flags().String("keymap", "", "keybinding preset (default|vim|emacs)")

	serveCmd.Flags().IntP("port", "p", 0, "server port")
	serveCmd.Flags().Bool("open", false, "open browser after starting")
	serveCmd.Flags().Bool("no-https", false, "disable HTTPS even if certs exist")

	exportCmd.Flags().StringP("format", "f", "html", "export format (html|pdf)")
	exportCmd.Flags().StringP("output", "o", "", "output directory")

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
