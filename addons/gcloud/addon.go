package gcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	defaultProject   string
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

func WithDefaultProject(projectID string) option {
	return func(c *config) {
		c.defaultProject = projectID
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
	if addon.Config.defaultProject != "" {
		if _, err := shx.Run(
			ctx,
			[]string{"gcloud", "config", "set", "project", addon.Config.defaultProject},
			shx.PassStdio(),
			shx.WithCombinedError(),
		); err != nil {
			return err
		}
	}
	if addon.Config.skipLogin {
		return nil
	}
	return LoginUser(ctx)
}

// LoginUser runs gcloud login steps if necessary.
//
// If allowedDomains is nil, it will use the addon's configured default setting.
// If it is not nill but empty, then it will accept any already logged in
// account as sufficient. Otherwise it will only skip the login if an active
// acccount in one of the given domains is found.
func LoginUser(ctx context.Context) error {
	// check current accounts
	accounts, err := getAccounts(ctx)
	if err != nil {
		return err
	}
	// see if any active account in an allowed domain is present
	if _, err := activeAccount(accounts, addon.Config.allowedDomains); err == nil {
		if err := copyADC(ctx); err != nil {
			return err
		}
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

func LoginServiceAccount(ctx context.Context, email string) (finalErr error) {
	if !strings.HasSuffix(email, ".iam.gserviceaccount.com") {
		return fmt.Errorf("service email must be in .iam.gserviceaccount.com domain")
	}
	accounts, err := getAccounts(ctx)
	if err != nil {
		return err
	}
	if acct, err := activeAccount(accounts, nil); err == nil && acct.Account == email {
		fmt.Println("gcloud already logged in the desired service account")
		return nil
	}

	// creating a service account key requires us to log in as a user, use that
	// user to create the key, and then revoke that user login

	fmt.Println("Temporarily logging into gcloud as user to create service account key")
	if _, err := shx.Run(
		ctx,
		[]string{"gcloud", "auth", "login", "--update-adc"},
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}
	if accounts, err = getAccounts(ctx); err != nil {
		return err
	}
	userAccount, err := activeAccount(accounts, addon.Config.allowedDomains)
	if err != nil {
		return fmt.Errorf("failed to find active user account after login: %w", err)
	}
	// only revoke once, but be sure we do so regardless of why we leave
	revoked := false
	revoke := func() error {
		fmt.Println("Revoking temporary user gcloud login")
		if _, err := shx.Run(
			ctx,
			[]string{"gcloud", "auth", "revoke", userAccount.Account},
			shx.PassStdio(),
			shx.WithCombinedError(),
		); err != nil {
			return err
		}
		return nil
	}
	defer func() {
		if !revoked {
			if err := revoke(); err != nil {
				if finalErr == nil {
					finalErr = err
				} else {
					finalErr = errors.Join(finalErr, err)
				}
			}
		}
	}()

	fmt.Println("Creating service account key")
	td, err := os.MkdirTemp("", "gdev-svc-key.*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(td) //nolint:errcheck
	kp := filepath.Join(td, "key.json")
	// FUTURE: we could do this with the API SDK instead
	if _, err := shx.Run(
		ctx,
		[]string{"gcloud", "iam", "service-accounts", "keys", "create", kp, "--iam-account", email},
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}
	fmt.Println("Logging into gcloud as service account")
	if _, err := shx.Run(
		ctx,
		[]string{"gcloud", "auth", "activate-service-account", "--key-file", kp},
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}
	if err := copyADC(ctx); err != nil {
		return err
	}
	revoked = true
	if err := revoke(); err != nil {
		return err
	}
	return nil
}

func copyADC(ctx context.Context) error {
	accounts, err := getAccounts(ctx)
	if err != nil {
		return err
	}
	account, err := activeAccount(accounts, nil)
	if err != nil {
		return err
	}

	// gcloud saves an ADC file for each account, we can just copy it
	adcPath := filepath.Join(
		shx.HomeDir(),
		".config",
		"gcloud",
		"legacy_credentials",
		account.Account,
		"adc.json",
	)
	adcContents, err := os.ReadFile(adcPath)
	if err != nil {
		return err
	}
	// temp file dance
	gcDir := filepath.Join(shx.HomeDir(), ".config", "gcloud")
	tf, err := os.CreateTemp(gcDir, "gdev-adc.json.*")
	if err != nil {
		return err
	}
	defer os.Remove(tf.Name()) //nolint:errcheck
	defer tf.Close()           //nolint:errcheck
	if _, err := tf.Write(adcContents); err != nil {
		return err
	}
	if err := tf.Sync(); err != nil {
		return err
	}
	if err := tf.Close(); err != nil {
		return err
	}
	if err := os.Rename(tf.Name(), filepath.Join(gcDir, "application_default_credentials.json")); err != nil {
		return err
	}
	return nil
}

type gcloudAccount struct {
	Account string `json:"account"`
	Status  string `json:"status"`
}

func getAccounts(ctx context.Context) ([]gcloudAccount, error) {
	res, err := shx.Run(
		ctx,
		[]string{"gcloud", "auth", "list", "--format=json"},
		shx.CaptureOutput(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return nil, err
	}
	var accounts []gcloudAccount
	if err := json.NewDecoder(res.Stdout()).Decode(&accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func activeAccount(accounts []gcloudAccount, allowedDomains []string) (gcloudAccount, error) {
	for _, acct := range accounts {
		if acct.Status != "ACTIVE" {
			continue
		}
		if len(allowedDomains) == 0 {
			return acct, nil
		}
		_, domain, ok := strings.Cut(acct.Account, "@")
		if !ok {
			return gcloudAccount{}, fmt.Errorf("invalid account email, no @: %q", acct.Account)
		}
		if slices.Contains(allowedDomains, domain) {
			return acct, nil
		}
	}
	return gcloudAccount{}, errNoActiveAccount
}

var errNoActiveAccount = fmt.Errorf("no active account in valid domain found")
