package tailscale

import (
	_ "embed"
	"fmt"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	apt_common "fastcat.org/go/gdev/addons/bootstrap/apt/common"
	"fastcat.org/go/gdev/lib/shx"
)

// addon describes the addon provided by this package.
//
// Do NOT export this variable.
var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "tailscale",
		Description: func() string {
			return "Tailscale VPN client"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	skipLogin bool
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

func WithSkipLogin() option {
	return func(c *config) {
		c.skipLogin = true
	}
}

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(
		bootstrap.WithSteps(apt.PublicSourceInstallSteps(
			&apt.SourceInstaller{
				SourceName: "tailscale",
				Source: &apt.Source{
					// values will be updated via RuntimeUpdate to match the observed OS info
					Types:      []string{"deb"},
					URIs:       []string{"https://pkgs.tailscale.com/stable/debian"},
					Suites:     []string{"trixie"},
					Components: []string{"main"},
					SignedBy:   "/usr/share/keyrings/tailscale-archive-keyring.gpg",
				},
				SigningKey: tailScaleAptKeyring,
				// be compatible with what their installer does
				Deb822: false,
				RuntimeUpdate: func(si *apt.SourceInstaller) error {
					// switch the URI & suite to the proper value based on the os release
					osInfo, err := apt_common.HostOSRelease()
					if err != nil {
						return fmt.Errorf("failed to get OS info for tailscale apt source: %w", err)
					}
					si.Source.URIs = []string{fmt.Sprintf("https://pkgs.tailscale.com/stable/%s", osInfo.ID)}
					// codename is in osInfo.Extra, but logic to extract is not worth repeating
					si.Source.Suites = []string{apt_common.HostOSVersionCodename()}
					return nil
				},
			},
		)...),
		apt.WithPackages("Select tailscale packages",
			"tailscale",
			"tailscale-archive-keyring",
		),
		bootstrap.WithSteps(bootstrap.NewStep(
			ConfigureStepName,
			configureTailscale,
			bootstrap.AfterSteps(apt.StepNameInstall),
		)),
	)
})

const ConfigureStepName = "Configure tailscale"

func configureTailscale(ctx *bootstrap.Context) error {
	if addon.Config.skipLogin {
		return nil
	}
	return TailscaleUp(ctx)
}

func TailscaleUp(ctx *bootstrap.Context) error {
	fmt.Println("Bringing up tailscale...")
	_, err := shx.Run(
		ctx,
		[]string{"tailscale", "up"},
		shx.PassStdio(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return fmt.Errorf("failed to bring up tailscale: %w", err)
	}
	return nil
}

// TODO: this needs to be more dynamic in principle, if the keyrings vary by
// archive. As of 2025-10-17, they all seem to be the same.
//
//go:generate go tool getkey https://pkgs.tailscale.com/stable/debian/trixie.noarmor.gpg tailscale-archive-keyring.gpg
//go:embed tailscale-archive-keyring.gpg
var tailScaleAptKeyring []byte
