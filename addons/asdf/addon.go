package asdf

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/addons/bootstrap/textedit"
	"fastcat.org/go/gdev/addons/github"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/shx"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "asdf",
		Description: func() string {
			return "asdf tool version manager support"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{
		// Initialize your addon configuration here
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	plugins []string
	tools   []Tool

	// alternate files in which to place the snippet to add the asdf shims to PATH
	// and what type of snippet to put there. If unset, will put a posix snippet
	// in `~/.profile`.
	shimsFiles map[string]string
	// alternate files in which to place the snippet to add the asdf shell
	// completion. If unset, will put bash completion in `~/.bashrc`. Keys are
	// filenames, values are the completion type.
	completionFiles map[string]string
}

type Tool struct {
	Name        string
	Version     string
	MakeDefault bool
}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBootstrap()

	addon.RegisterIfNeeded()
}

func initialize() error {
	return nil
}

func WithPlugins(plugins ...string) option {
	return func(c *config) {
		c.plugins = append(c.plugins, plugins...)
	}
}

func WithTool(name, version string) option {
	return func(c *config) {
		c.tools = append(c.tools, Tool{Name: name, Version: version})
	}
}

func WithDefaultTool(name, version string) option {
	return func(c *config) {
		c.tools = append(c.tools, Tool{Name: name, Version: version, MakeDefault: true})
	}
}

func WithTools(tools ...Tool) option {
	for _, t := range tools {
		if t.Name == "" || t.Version == "" {
			panic("asdf tool must have a name and version")
		}
	}
	return func(c *config) {
		c.tools = append(c.tools, tools...)
	}
}

// Changes the shell files where the asdf shims path snippet is added.
//
// By default, the posix shell snippet is added to `~/.profile`. If this option
// is used, it overrides that, so if you want to add it to additional places you
// must add an entry for `~/.profile` of type "sh".
//
// A leading `~/` is expanded to the user's home directory.
//
// Otherwise all paths must be absolute.
func WithShimsFile(file, syntax string) option {
	if strings.HasPrefix(file, "~/") {
		file = filepath.Join(shx.HomeDir(), file[2:])
	}
	if !filepath.IsAbs(file) {
		panic(fmt.Errorf("asdf shims file paths must be absolute: %q", file))
	}
	switch syntax {
	case "bash", "zsh":
		syntax = "sh"
	case "sh", "fish", "elvish", "powershell", "nushell":
		// ok
	default:
		panic(fmt.Errorf("unsupported asdf shims file syntax: %q", syntax))
	}
	return func(c *config) {
		c.shimsFiles[file] = syntax
	}
}

// Changes the shell files where the asdf completion snippet is added.
//
// By default, bash completion is added to `~/.bashrc`. If this option is used,
// it overrides that, so if you want to add it to additional places without
// removing the bashrc one, you must make a call to re-add the bashrc entry.
//
// A leading `~/` is expanded to the user's home directory.
//
// Otherwise all paths must be absolute.
func WithCompletionFile(file, completionType string) option {
	if strings.HasPrefix(file, "~/") {
		file = filepath.Join(shx.HomeDir(), file[2:])
	}
	if !filepath.IsAbs(file) {
		panic(fmt.Errorf("asdf completion file paths must be absolute: %q", file))
	}
	switch completionType {
	// supported completions taken from asdf v0.18.0:
	case "bash", "elvish", "fish", "nushell", "zsh":
	default:
		panic(fmt.Errorf("unsupported asdf completion type: %q", completionType))
	}
	return func(c *config) {
		if c.completionFiles == nil {
			c.completionFiles = make(map[string]string)
		}
		c.completionFiles[file] = completionType
	}
}

var configureBootstrap = sync.OnceFunc(func() {
	const installName = "Install asdf"
	const pluginsName = "Install asdf plugins"
	const toolsName = "Install asdf tools"
	const configsName = "Configure asdf tool defaults"
	const shimsName = "Configure asdf in shell"
	const depsName = "Select asdf completion dependencies"
	bootstrap.Configure(bootstrap.WithSteps(
		bootstrap.NewStep(
			installName,
			installAsdf,
		),
		bootstrap.NewStep(
			pluginsName,
			installPlugins,
			// plugin install needs git
			bootstrap.AfterSteps(installName, apt.StepNameInstall),
		),
		bootstrap.NewStep(
			toolsName,
			installTools,
			bootstrap.AfterSteps(pluginsName),
		),
		bootstrap.NewStep(
			configsName,
			configureTools,
			bootstrap.AfterSteps(toolsName),
		),
		bootstrap.NewStep(
			shimsName,
			configureShell,
			bootstrap.AfterSteps(installName),
		),
		bootstrap.NewStep(
			depsName,
			func(ctx *bootstrap.Context) error {
				if len(addon.Config.completionFiles) == 0 ||
					internal.SeqContains(maps.Values(addon.Config.completionFiles), "bash") {
					apt.AddPackages(ctx, "bash-completion")
				}
				return nil
			},
			bootstrap.BeforeSteps(apt.StepNameInstall),
		),
	))
})

func installAsdf(ctx *bootstrap.Context) error {
	fmt.Println("Installing asdf version manager...")
	// FUTURE: consider `go install` instead of trusting upstream tarballs
	// TODO: check installed version and at least print some info, maybe skip
	// upgrade if latest version is already installed.

	ghc := github.NewClient()
	rel, err := ghc.GetRelease(ctx, "asdf-vm", "asdf", "latest")
	if err != nil {
		return fmt.Errorf("failed to fetch asdf release: %w", err)
	}
	// e.g. asdf-v0.18.0-linux-amd64.tar.gz
	assetName := "asdf-" + rel.TagName + "-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	i := slices.IndexFunc(rel.Assets, func(a github.ReleaseAsset) bool { return a.Name == assetName })
	if i < 0 {
		return fmt.Errorf("asdf release %s has no asset %s", rel.TagName, assetName)
	}
	resp, err := ghc.Download(ctx, rel.Assets[i].URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// download to a temp file
	tf, err := os.CreateTemp("", instance.AppName()+"-asdf-*")
	if err != nil {
		return err
	}
	defer tf.Close() // nolint:errcheck
	tfn := tf.Name()
	defer os.Remove(tfn) // nolint:errcheck

	// expect a tar.gz file with exactly one file in it named `asdf`
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("asdf download %s corrupt: %w", rel.Assets[i].URL, err)
	}
	tr := tar.NewReader(zr)
	th, err := tr.Next()
	if err != nil {
		return fmt.Errorf("asdf download %s corrupt: %w", rel.Assets[i].URL, err)
	}
	if th.Name != "asdf" {
		return fmt.Errorf(
			"asdf download %s corrupt: expected single file named 'asdf', got %s",
			rel.Assets[i].URL,
			th.Name,
		)
	}
	if _, err := io.Copy(tf, tr); err != nil {
		return fmt.Errorf("failed to extract asdf binary: %w", err)
	} else if err := tf.Close(); err != nil {
		return fmt.Errorf("failed to flush asdf temp file %s: %w", tfn, err)
	} else if err := os.Chmod(tf.Name(), 0o755); err != nil {
		return fmt.Errorf("failed to make asdf executable: %w", err)
	}

	// have to finish reading things out for the gzip checksum verification to work
	if _, err := tr.Next(); err != io.EOF {
		return fmt.Errorf("asdf download %s corrupt: expected single file, got more than one", rel.Assets[i].URL)
	} else if err := zr.Close(); err != nil {
		return fmt.Errorf("asdf download %s corrupt: %w", rel.Assets[i].URL, err)
	}

	// TODO: this assumes that ~/.local/bin/ is in the user's PATH, or at least
	// will be once they reboot if it wasn't added because it didn't exist yet.
	destDir := filepath.Join(shx.HomeDir(), ".local", "bin")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create asdf destination directory %s: %w", destDir, err)
	}
	if err := os.Rename(tfn, filepath.Join(destDir, "asdf")); err != nil {
		return fmt.Errorf("failed to install asdf binary to %s: %w", destDir, err)
	}

	// ~/.local/bin might not be in the PATH because most bashrc setups only add
	// it if it exists. Make sure it's there now so we can run the just-installed
	// copy.
	// TODO: put this in shx or something
	if slices.Index(filepath.SplitList(os.Getenv("PATH")), destDir) < 0 {
		if err := os.Setenv("PATH", destDir+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
			return fmt.Errorf("failed to add %s to PATH: %w", destDir, err)
		}
	}

	return nil
}

func installPlugins(ctx *bootstrap.Context) error {
	if len(addon.Config.plugins) == 0 {
		return nil
	}
	fmt.Printf("Installing %d asdf plugins...\n", len(addon.Config.plugins))

	for _, plugin := range addon.Config.plugins {
		fmt.Printf("Installing asdf plugin %s...\n", plugin)
		if _, err := shx.Run(
			ctx,
			[]string{"asdf", "plugin", "add", plugin},
			shx.PassOutput(),
			shx.WithCombinedError(),
		); err != nil {
			return fmt.Errorf("failed to install asdf plugin %s: %w", plugin, err)
		}
	}

	return nil
}

func installTools(ctx *bootstrap.Context) error {
	if len(addon.Config.tools) == 0 {
		return nil
	}
	fmt.Printf("Installing %d asdf tools...\n", len(addon.Config.tools))

	for _, tool := range addon.Config.tools {
		fmt.Printf("Installing asdf tool %s version %s...\n", tool.Name, tool.Version)
		if _, err := shx.Run(
			ctx,
			[]string{"asdf", "install", tool.Name, tool.Version},
			shx.PassOutput(),
			shx.WithCombinedError(),
		); err != nil {
			return fmt.Errorf("failed to install asdf tool %s version %s: %w", tool.Name, tool.Version, err)
		}
	}

	return nil
}

func configureTools(ctx *bootstrap.Context) error {
	defaults := internal.FilterSlice(addon.Config.tools, func(t Tool) bool { return t.MakeDefault })
	if len(defaults) == 0 {
		return nil
	}
	fmt.Printf("Configuring %d asdf tools as defaults...\n", len(defaults))
	for _, tool := range defaults {
		fmt.Printf("Configuring asdf tool %s version %s as default...\n", tool.Name, tool.Version)
		if _, err := shx.Run(
			ctx,
			[]string{"asdf", "set", "--home", tool.Name, tool.Version},
			shx.PassOutput(),
			shx.WithCombinedError(),
		); err != nil {
			return fmt.Errorf(
				"failed to configure asdf tool %s version %s as default: %w",
				tool.Name, tool.Version, err,
			)
		}
	}
	return nil
}

func configureShell(ctx *bootstrap.Context) error {
	if err := configureShims(ctx); err != nil {
		return err
	}
	if err := configureCompletion(ctx); err != nil {
		return err
	}
	return nil
}

func configureShims(ctx *bootstrap.Context) error {
	files := addon.Config.shimsFiles
	if len(files) == 0 {
		files = map[string]string{filepath.Join(shx.HomeDir(), ".profile"): "sh"}
	}
	for file, syntax := range files {
		var editor textedit.Editor
		switch syntax {
		case "sh":
			editor = textedit.SpliceRange(
				fmt.Sprintf("# %s: setup asdf shims in PATH", instance.AppName()),
				// NOTE: this is sourced via dash, so must be _strictly_ POSIX sh, no bashisms!
				// this is a fancier snippet than the asdf docs provide
				`_asdfshims() {`, //cspell:ignore asdfshims
				`	local shims="${ASDF_DATA_DIR:-$HOME/.asdf}/shims"`,
				`	case "$PATH" in`,
				`		"$shims":*|*:"$shims":*|*:"$shims") ;;`,
				`		*) export PATH="$shims:$PATH"`,
				`	esac`, //cspell:ignore esac
				`}`,
				`_asdfshims`,
				`unset -f _asdfshims`,
				fmt.Sprintf("# %s end asdf shims setup", instance.AppName()),
			)
		default:
			return fmt.Errorf("asdf shims file syntax %s not yet implemented", syntax)
		}
		fmt.Printf("Configuring asdf shims in %s...\n", file)
		if changed, err := textedit.EditFile(file, editor); err != nil {
			return fmt.Errorf("failed to configure asdf shims in %s: %w", file, err)
		} else if changed {
			// modifying the user's shell rc files often requires a reboot to take full effect
			bootstrap.SetNeedsReboot(ctx)
		}
	}
	return nil
}

func configureCompletion(ctx *bootstrap.Context) error {
	filesMap := addon.Config.completionFiles
	if len(filesMap) == 0 {
		filesMap = map[string]string{filepath.Join(shx.HomeDir(), ".bashrc"): "bash"}
	}
	for f, completionType := range filesMap {
		var editor textedit.Editor
		switch completionType {
		case "bash":
			// bashrc is often sourced before ~/.local/bin is added to PATH, so we
			// include the full path here. While the snippet is just a single line, we
			// use the range markers in case we change the format of this path hack.
			asdfPath := filepath.Join(shx.HomeDir(), ".local", "bin", "asdf")
			editor = textedit.SpliceRange(
				fmt.Sprintf("# %s: setup asdf completion", instance.AppName()),
				fmt.Sprintf(". <(%s completion bash)", asdfPath),
				fmt.Sprintf("# %s end asdf completion setup", instance.AppName()),
			)
		default:
			return fmt.Errorf("asdf %s completion not yet implemented", completionType)
		}
		fmt.Printf("Configuring asdf %s completion in %s...\n", completionType, f)
		if changed, err := textedit.EditFile(f, editor); err != nil {
			return fmt.Errorf("failed to configure asdf %s completion in %s: %w", completionType, f, err)
		} else if changed {
			bootstrap.SetNeedsReboot(ctx)
		}
	}
	return nil
}
