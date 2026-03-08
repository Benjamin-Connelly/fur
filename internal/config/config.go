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
}

// ServerConfig holds web server settings.
type ServerConfig struct {
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
	NoHTTPS bool   `mapstructure:"no_https"`
	Open    bool   `mapstructure:"open"`
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

	v.SetEnvPrefix("LOOKIT")
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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

// CreateDefault writes a default config file to ~/.config/lookit/config.yaml
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

	content := `# Lookit configuration
# See: https://github.com/Benjamin-Connelly/lookit

# Theme: light, dark, or auto
theme: auto

# Keybinding preset: default, vim, or emacs
keymap: default

# Web server settings
server:
  port: 7777
  host: localhost
  no_https: false
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

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing config: %w", err)
	}

	return configPath, nil
}

// ConfigDir returns the lookit configuration directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lookit"), nil
}
