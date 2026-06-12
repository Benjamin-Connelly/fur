package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"go.yaml.in/yaml/v3"
)

// ownedByCurrentUser reports whether info is owned by the current uid. On
// platforms whose FileInfo.Sys() is not a *syscall.Stat_t (i.e. non-POSIX) it
// returns true and the caller relies on the write-permission bits alone. fur
// targets linux/darwin, where Stat_t is available.
func ownedByCurrentUser(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	return int(st.Uid) == os.Getuid()
}

// HookPoint identifies when a plugin hook fires.
type HookPoint int

const (
	HookBeforeRender HookPoint = iota
	HookAfterRender
	HookBeforeIndex
	HookAfterIndex
	HookOnNavigate
)

// Hook is a function called at a specific point in processing.
type Hook struct {
	Name  string
	Point HookPoint
	Fn    func(ctx *HookContext) error
}

// HookContext provides data to hook functions.
type HookContext struct {
	FilePath string
	Content  string
	Metadata map[string]interface{}
	Theme    string
	Width    int
	Format   string
}

// PluginConfig represents a YAML plugin definition.
type PluginConfig struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Hooks       []HookConfig `yaml:"hooks"`
}

// HookConfig defines a hook in a YAML plugin file. Hooks are content
// transforms only — prepend, append, and replace. There is deliberately no
// command/exec field: plugin files live in the user's config dir and are
// loaded automatically, so an exec hook would be a code-execution sink fed by
// any file that lands in ~/.config/fur/plugins. See lookit-9py.4.1.
type HookConfig struct {
	Point   string        `yaml:"point"`
	Prepend string        `yaml:"prepend"`
	Append  string        `yaml:"append"`
	Replace []ReplaceRule `yaml:"replace"`
}

// ReplaceRule defines a string replacement in content.
type ReplaceRule struct {
	Old string `yaml:"old"`
	New string `yaml:"new"`
}

// Registry manages registered plugin hooks.
type Registry struct {
	hooks map[HookPoint][]Hook
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		hooks: make(map[HookPoint][]Hook),
	}
}

// Register adds a hook at the specified point.
func (r *Registry) Register(hook Hook) {
	r.hooks[hook.Point] = append(r.hooks[hook.Point], hook)
}

// Run executes all hooks registered at the given point.
// For HookBeforeRender, the Content field may be modified by hooks.
func (r *Registry) Run(point HookPoint, ctx *HookContext) error {
	for _, hook := range r.hooks[point] {
		if err := hook.Fn(ctx); err != nil {
			return fmt.Errorf("hook %q: %w", hook.Name, err)
		}
	}
	return nil
}

// LoadPlugins scans configDir for .yaml plugin definitions and registers their hooks.
func LoadPlugins(configDir string) (*Registry, error) {
	registry := NewRegistry()

	pluginDir := filepath.Join(configDir, "plugins")
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return nil, fmt.Errorf("reading plugin dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(pluginDir, entry.Name())

		// Trust gate: plugin hooks inject content into every rendered document
		// (and thus into the terminal / HTML pipeline). Only honor plugin files
		// that the current user owns and that are not writable by group or
		// other — a file a co-located adversary can write is not trustworthy.
		if !trustedPluginFile(path) {
			fmt.Fprintf(os.Stderr, "warning: skipping plugin %s: not owner-only (group/other-writable or foreign owner)\n", entry.Name())
			continue
		}

		if err := loadPluginFile(registry, path); err != nil {
			return nil, fmt.Errorf("loading plugin %s: %w", entry.Name(), err)
		}
	}

	return registry, nil
}

// trustedPluginFile reports whether a plugin file is safe to load: it must be
// a regular file, owned by the current user, and not writable by group or
// other. On platforms without POSIX ownership the owner check is skipped and
// only the write-permission bits are enforced. Stat errors fail closed.
func trustedPluginFile(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if !info.Mode().IsRegular() {
		return false // symlinks and irregular files are not trusted
	}
	if info.Mode().Perm()&0o022 != 0 {
		return false // group- or world-writable
	}
	if !ownedByCurrentUser(info) {
		return false
	}
	return true
}

func loadPluginFile(registry *Registry, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg PluginConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	for _, hc := range cfg.Hooks {
		point, err := parseHookPoint(hc.Point)
		if err != nil {
			return err
		}

		hookCfg := hc // capture for closure
		hook := Hook{
			Name:  fmt.Sprintf("%s:%s", cfg.Name, hc.Point),
			Point: point,
			Fn:    makeHookFn(hookCfg),
		}
		registry.Register(hook)
	}

	return nil
}

func makeHookFn(hc HookConfig) func(ctx *HookContext) error {
	return func(ctx *HookContext) error {
		// Prepend content
		if hc.Prepend != "" {
			ctx.Content = hc.Prepend + ctx.Content
		}

		// Apply replacements
		for _, r := range hc.Replace {
			ctx.Content = strings.ReplaceAll(ctx.Content, r.Old, r.New)
		}

		// Append content
		if hc.Append != "" {
			ctx.Content += hc.Append
		}

		return nil
	}
}

func parseHookPoint(s string) (HookPoint, error) {
	switch strings.ToLower(s) {
	case "beforerender", "before_render":
		return HookBeforeRender, nil
	case "afterrender", "after_render":
		return HookAfterRender, nil
	case "beforeindex", "before_index":
		return HookBeforeIndex, nil
	case "afterindex", "after_index":
		return HookAfterIndex, nil
	case "onnavigate", "on_navigate":
		return HookOnNavigate, nil
	default:
		return 0, fmt.Errorf("unknown hook point: %s", s)
	}
}
