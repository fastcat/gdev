package gcloud

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	apt_common "fastcat.org/go/gdev/addons/bootstrap/apt/common"
	"fastcat.org/go/gdev/shx"
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
	allowedDomains []string
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

func WithAllowedDomains(domains ...string) option {
	return func(c *config) {
		c.allowedDomains = domains
	}
}

var configureBootstrap = sync.OnceFunc(func() {
	// TODO: allow customization of the bootstrap steps somehow?
	// We can't do the bootstrap config in initialize(), so it gets a little tricky
	bootstrap.Configure(
		bootstrap.WithSteps(apt.PublicSourceInstallSteps(apt_common.GoogleCloudInstaller())...),
		apt.WithPackages("Select gcloud packages", "google-cloud-cli"),
		bootstrap.WithSteps(bootstrap.NewStep(
			configureGcloudStepName,
			configureGcloud,
			bootstrap.AfterSteps(apt.StepNameInstall),
		)),
	)
})

const configureGcloudStepName = "Configure gcloud"

func configureGcloud(ctx *bootstrap.Context) error {
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
		if len(addon.Config.allowedDomains) == 0 {
			loggedIn = true
			break
		}
		_, domain, ok := strings.Cut(acct.Account, "@")
		if !ok {
			return fmt.Errorf("invalid account email, no @: %q", acct.Account)
		}
		if slices.Contains(addon.Config.allowedDomains, domain) {
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

type gcloudAccount struct {
	Account string `json:"account"`
	Status  string `json:"status"`
}
