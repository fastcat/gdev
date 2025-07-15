module fastcat.org/go/gdev/magefiles

go 1.25

require (
	github.com/goccy/go-yaml v1.18.0
	github.com/magefile/mage v1.15.0
	golang.org/x/mod v0.26.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/telemetry v0.0.0-20250710130107-8d8967aff50b // indirect
	golang.org/x/tools v0.34.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
)

tool (
	github.com/magefile/mage
	golang.org/x/tools/cmd/stringer
	golang.org/x/vuln/cmd/govulncheck
)
