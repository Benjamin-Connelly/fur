package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	// Root directory to browse
	Root string `mapstructure:"root"`

	// Theme: light, dark, or auto
	Theme string `mapstructure:"theme"`

	// Keymap preset: default, vim, emacs
	Keymap string `mapstructure:"keymap"`

	// Web server settings
	Server ServerConfig `mapstructure:"server"`

	// Git integration settings
	Git GitConfig `mapstructure:"git"`

	// File patterns to ignore (in addition to .gitignore)
	Ignore []string `mapstructure:"ignore"`

	// Mouse enables mouse wheel scrolling in TUI
	Mouse bool `mapstructure:"mouse"`

	// ReadingGuide shows a full-row highlight on the cursor line
	ReadingGuide bool `mapstructure:"reading_guide"`

	// ScrollOff keeps this many lines visible above/below the cursor
	ScrollOff int `mapstructure:"scrolloff"`

	// Debug enables verbose logging
	Debug bool `mapstructure:"debug"`

	// ShowHidden surfaces dotfiles and dotdirs in listings, search, and the
	// link graph. Internal VCS directories (.git/.hg/.svn/.bzr) stay filtered
	// regardless because browsing them is never useful here.
	ShowHidden bool `mapstructure:"show_hidden"`

	// Remote holds named remote host configurations
	Remotes map[string]RemoteConfig `mapstructure:"remotes"`
}

// RemoteConfig defines a named remote host for SSH browsing.
type RemoteConfig struct {
	Host string `mapstructure:"host"`
	User string `mapstructure:"user"`
	Port int    `mapstructure:"port"`
	Path string `mapstructure:"path"`
}

// ServerConfig holds web server settings.
type ServerConfig struct {
	Port      int    `mapstructure:"port"`
	Host      string `mapstructure:"host"`
	NoHTTPS   bool   `mapstructure:"no_https"`
	Open      bool   `mapstructure:"open"`
	CustomCSS string `mapstructure:"custom_css"`
}

// GitConfig holds git integration settings.
type GitConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	ShowStatus bool   `mapstructure:"show_status"`
	Remote     string `mapstructure:"remote"`
}

var validThemes = map[string]bool{"light": true, "dark": true, "auto": true, "ascii": true}
var validKeymaps = map[string]bool{"default": true, "vim": true, "emacs": true}

// Validate checks that config values are within allowed ranges.
func (c *Config) Validate() error {
	if !validThemes[c.Theme] {
		return fmt.Errorf("invalid theme %q: must be light, dark, or auto", c.Theme)
	}
	if !validKeymaps[c.Keymap] {
		return fmt.Errorf("invalid keymap %q: must be default, vim, or emacs", c.Keymap)
	}
	return nil
}

// String returns a human-readable representation of the config for debugging.
func (c *Config) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "root:       %s\n", c.Root)
	fmt.Fprintf(&b, "theme:      %s\n", c.Theme)
	fmt.Fprintf(&b, "keymap:     %s\n", c.Keymap)
	fmt.Fprintf(&b, "server:     %s:%d (https=%t, open=%t)\n",
		c.Server.Host, c.Server.Port, !c.Server.NoHTTPS, c.Server.Open)
	fmt.Fprintf(&b, "git:        enabled=%t status=%t remote=%s\n",
		c.Git.Enabled, c.Git.ShowStatus, c.Git.Remote)
	fmt.Fprintf(&b, "show_hidden: %t\n", c.ShowHidden)
	if len(c.Ignore) > 0 {
		fmt.Fprintf(&b, "ignore:     %s\n", strings.Join(c.Ignore, ", "))
	}
	return b.String()
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Root:      ".",
		Theme:     "auto",
		Keymap:    "default",
		ScrollOff: 5,
		Server: ServerConfig{
			Port: 7777,
			Host: "localhost",
		},
		Git: GitConfig{
			Enabled:    true,
			ShowStatus: true,
			Remote:     "origin",
		},
	}
}

// Load reads configuration from file and environment, merging with defaults.
func Load(cfgFile string) (*Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		configDir, err := ConfigDir()
		if err != nil {
			return cfg, nil // Use defaults if no config dir
		}
		v.AddConfigPath(configDir)
	}

	v.SetEnvPrefix("FUR")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return cfg, nil // No config file is fine
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Merge per-project config (walks up from CWD)
	mergeProjectConfig(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// mergeProjectConfig walks up from CWD looking for .fur.toml or .fur.yaml
// and merges any found settings into the config. Project config overrides global.
func mergeProjectConfig(cfg *Config) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		for _, name := range []string{".fur.toml", ".fur.yaml", ".fur.yml"} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			v := viper.New()
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err != nil {
				continue
			}
			if err := v.Unmarshal(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "warning: parsing project config %s: %v\n", path, err)
			}
			return // Use first found, stop walking
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// Watch starts watching the config file for changes and reloads on modification.
// The onChange callback is invoked with the new config after each successful reload.
func Watch(cfgFile string, onChange func(*Config)) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		configDir, err := ConfigDir()
		if err != nil {
			return
		}
		v.AddConfigPath(configDir)
	}

	_ = v.ReadInConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		cfg := DefaultConfig()
		if err := v.Unmarshal(cfg); err != nil {
			return
		}
		if err := cfg.Validate(); err != nil {
			return
		}
		if onChange != nil {
			onChange(cfg)
		}
	})
	v.WatchConfig()
}

// DefaultConfigYAML is the starter config.yaml content written by
// CreateDefault and 'fur config init'. Exported so tests and the CLI
// can share the same canonical template.
const DefaultConfigYAML = `# fur configuration
# See: https://github.com/Benjamin-Connelly/fur

# Theme: light, dark, auto, or ascii
theme: auto

# Keybinding preset: default, vim, or emacs
keymap: default

# Show dotfiles and dotdirs (e.g. .gitignore, .github/) in listings.
# .git/.hg/.svn/.bzr stay hidden regardless.
show_hidden: false

# Web server settings
server:
  port: 7777
  host: localhost
  no_https: false
  # Auto-open the browser when 'fur serve' starts. Skipped under SSH.
  open: false

# Git integration
git:
  enabled: true
  show_status: true
  remote: origin

# Additional ignore patterns (beyond .gitignore)
# ignore:
#   - "*.tmp"
#   - "vendor/"
`

// CreateDefault writes a default config file to ~/.config/fur/config.yaml
// if one does not already exist. Returns the path written, or empty string if
// a config already exists.
func CreateDefault() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configPath); err == nil {
		return "", nil // already exists
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("creating config dir: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(DefaultConfigYAML), 0o644); err != nil {
		return "", fmt.Errorf("writing config: %w", err)
	}

	return configPath, nil
}

// CreateDefaultForce writes the default config to ~/.config/fur/config.yaml,
// overwriting any existing file. If a config already exists, it is renamed
// to config.yaml.bak (clobbering an older .bak) before the new file is
// written. Returns (newPath, backupPath). backupPath is empty when no prior
// config existed.
func CreateDefaultForce() (string, string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("determining config dir: %w", err)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", "", fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	backupPath := ""

	if _, err := os.Stat(configPath); err == nil {
		backupPath = configPath + ".bak"
		if err := os.Rename(configPath, backupPath); err != nil {
			return "", "", fmt.Errorf("backing up existing config: %w", err)
		}
	}

	if err := os.WriteFile(configPath, []byte(DefaultConfigYAML), 0o644); err != nil {
		return "", backupPath, fmt.Errorf("writing config: %w", err)
	}

	return configPath, backupPath, nil
}

// ConfigDir returns the fur configuration directory.
// Migrates from the legacy ~/.config/lookit path if it exists.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	newPath := filepath.Join(home, ".config", "fur")
	legacyPath := filepath.Join(home, ".config", "lookit")
	if _, err := os.Stat(legacyPath); err == nil {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "note: migrating config from %s to %s\n", legacyPath, newPath)
			if err := os.Rename(legacyPath, newPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: config migration failed: %v\n", err)
			}
		}
	}
	return newPath, nil
}
