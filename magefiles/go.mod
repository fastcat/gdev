module fastcat.org/go/gdev/magefiles

go 1.27.1

require (
	github.com/goccy/go-yaml v1.19.2
	github.com/magefile/mage v1.17.2
	golang.org/x/mod v0.42.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/telemetry v0.0.0-20260924152758-ed294f943157 // indirect
	golang.org/x/tools v0.51.0 // indirect
	golang.org/x/vuln v1.8.0 // indirect
)

tool (
	github.com/magefile/mage
	golang.org/x/tools/cmd/stringer
	golang.org/x/vuln/cmd/govulncheck
)
