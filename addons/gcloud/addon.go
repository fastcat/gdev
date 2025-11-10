package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/lib/shx"
)

// addon describes the addon provided by this package.
//
// Do NOT export this variable.
var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "gcloud",
		Description: func() string {
			return "gcloud addon to install and configure the Google Cloud CLI"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	skipLogin        bool
	includeTransport bool
	allowedDomains   []string
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

// WithSkipLogin causes the bootstrap sequence to no-op the login step.
func WithSkipLogin() option {
	return func(c *config) {
		if configuredBootstrap {
			panic("WithSkipLogin must be called before first Configure()")
		}
		c.skipLogin = true
	}
}

func WithAptTransport() option {
	return func(c *config) {
		if configuredBootstrap {
			panic("WithAptTransport must be called before first Configure()")
		}
		c.includeTransport = true
	}
}

func WithAllowedDomains(domains ...string) option {
	return func(c *config) {
		c.allowedDomains = domains
	}
}

var (
	configuredBootstrap bool
	configureBootstrap  = sync.OnceFunc(func() {
		sources := []*apt.SourceInstaller{CLISourceInstaller()}
		packages := []string{"google-cloud-cli"}
		if addon.Config.includeTransport {
			sources = append(sources, AptTransportSourceInstaller())
			packages = append(packages, "apt-transport-artifact-registry")
		}
		bootstrap.Configure(
			bootstrap.WithSteps(apt.PublicSourceInstallSteps(sources...)...),
			apt.WithPackages("Select gcloud packages", packages...),
			bootstrap.WithSteps(bootstrap.NewStep(
				ConfigureStepName,
				configureGcloud,
				bootstrap.AfterSteps(apt.StepNameInstall),
			)),
		)
		configuredBootstrap = true
	})
)

const ConfigureStepName = "Configure gcloud"

func configureGcloud(ctx *bootstrap.Context) error {
	if addon.Config.skipLogin {
		return nil
	}
	return LoginUser(ctx, addon.Config.allowedDomains)
}

// LoginUser runs gcloud login steps if necessary.
//
// If allowedDomains is nil, it will use the addon's configured default setting.
// If it is not nill but empty, then it will accept any already logged in
// account as sufficient. Otherwise it will only skip the login if an active
// acccount in one of the given domains is found.
func LoginUser(ctx context.Context, allowedDomains []string) error {
	if allowedDomains == nil {
		allowedDomains = addon.Config.allowedDomains
	}
	// check current accounts
	res, err := shx.Run(
		ctx,
		[]string{"gcloud", "auth", "list", "--format=json"},
		shx.CaptureOutput(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return err
	}
	var accounts []gcloudAccount
	if err := json.NewDecoder(res.Stdout()).Decode(&accounts); err != nil {
		return err
	}
	// see if any active account in an allowed domain is present
	loggedIn := false
	for _, acct := range accounts {
		if acct.Status != "ACTIVE" {
			continue
		}
		if len(allowedDomains) == 0 {
			loggedIn = true
			break
		}
		_, domain, ok := strings.Cut(acct.Account, "@")
		if !ok {
			return fmt.Errorf("invalid account email, no @: %q", acct.Account)
		}
		if slices.Contains(allowedDomains, domain) {
			loggedIn = true
			break
		}
	}
	if loggedIn {
		fmt.Println("gcloud already logged in")
		return nil
	}

	if _, err := shx.Run(
		ctx,
		[]string{"gcloud", "auth", "login", "--update-adc"},
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}

	// TODO: check that the user logged in to a permitted domain

	return nil
}

// TODO: LoginServiceAccount

type gcloudAccount struct {
	Account string `json:"account"`
	Status  string `json:"status"`
}
