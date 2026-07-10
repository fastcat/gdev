module fastcat.org/go/gdev/magefiles

go 1.26.4

require (
	github.com/goccy/go-yaml v1.19.2
	github.com/magefile/mage v1.17.2
	golang.org/x/mod v0.38.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260708182218-49f421fb7959 // indirect
	golang.org/x/tools v0.48.0 // indirect
	golang.org/x/vuln v1.5.0 // indirect
)

tool (
	github.com/magefile/mage
	golang.org/x/tools/cmd/stringer
	golang.org/x/vuln/cmd/govulncheck
)
