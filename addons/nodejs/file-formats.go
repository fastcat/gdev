package nodejs

// PackageJSON is a partial representation of a package.json file.
type PackageJSON struct {
	Name       string            `json:"name,omitempty"`
	Scripts    map[string]string `json:"scripts,omitempty"`
	Workspaces []string          `json:"workspaces,omitempty"`
}

// PNPMWorkspaceYAML is a partial representation of a pnpm-workspace.yaml file.
type PNPMWorkspaceYAML struct {
	Packages []string `json:"packages,omitempty"`
}

// RushJSON is a partial representation of a rush.json file.
type RushJSON struct {
	Projects []RushProject `json:"projects,omitempty,omitzero"`
}

type RushProject struct {
	PackageName   string `json:"packageName,omitempty"`
	ProjectFolder string `json:"projectFolder,omitempty"`
	// other fields omitted as not needed
}
