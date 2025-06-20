package nodejs

// PackageJSON is a partial representation of a package.json file.
type PackageJSON struct {
	Name       string            `json:"name,omitempty"`
	Scripts    map[string]string `json:"scripts,omitempty"`
	Workspaces []string          `json:"workspaces,omitempty"`
}

type PNPMWorkspacesYAML struct {
	Packages []string `json:"packages,omitempty"`
}
