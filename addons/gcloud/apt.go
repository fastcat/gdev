package gcloud

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// use a fragment to trick the code because while the url says .gpg, it returns an .asc file
//
//go:generate go tool getkey https://packages.cloud.google.com/apt/doc/apt-key.gpg#.asc google-cloud.asc
//go:embed google-cloud.asc
var GoogleCloudArchiveKeyring []byte

func CLISourceInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "google-cloud-sdk",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://packages.cloud.google.com/apt"},
			Suites:     []string{"cloud-sdk"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/cloud.google.asc",
		},
		SigningKey: GoogleCloudArchiveKeyring,
		Deb822:     true,
	}
}

func AptTransportSourceInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "google-cloud-apt",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://packages.cloud.google.com/apt"},
			Suites:     []string{"apt-transport-artifact-registry-stable"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/cloud.google.asc",
		},
		SigningKey: GoogleCloudArchiveKeyring,
		Deb822:     true,
	}
}

func InstallAptTransportSteps() []*bootstrap.Step {
	var s []*bootstrap.Step
	s = append(s, apt.PublicSourceInstallSteps(AptTransportSourceInstaller())...)
	s = append(s, apt.AddPackagesStep("Select AR apt transport", "apt-transport-artifact-registry"))
	return s
}

func ArtifactRegistryAptSource(
	location string,
	project string,
	repository string,
) *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: repository,
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"ar+https://" + location + "-apt.pkg.dev/projects/" + project},
			Suites:     []string{repository},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/cloud.google.asc",
		},
		SigningKey: GoogleCloudArchiveKeyring,
		Deb822:     true,
	}
}

// ArtifactRegistryAptSteps creates bootstrap steps to add a private Artifact
// Registry hosted APT source. It will order itself after the default gcloud
// login step, but you must order it before your package install with
// [bootstrap.BeforeSteps]. If you have used [WithSkipLogin], then you will need
// to include [bootstrap.AfterSteps] for the step in which you do the actual
// login.
func ArtifactRegistryAptSteps(
	location string,
	project string,
	repository string,
	opts ...bootstrap.StepOpt,
) []*bootstrap.Step {
	steps := []*bootstrap.Step{
		apt.SourceInstallStep(
			ArtifactRegistryAptSource(location, project, repository),
			// adding this apt source requires being logged in
			bootstrap.AfterSteps(VerifyStepName),
		),
	}
	for _, s := range steps {
		s.With(opts...)
	}
	return steps
}
