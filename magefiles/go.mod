module fastcat.org/go/gdev/magefiles

go 1.24.2

require github.com/magefile/mage v1.15.0

require (
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/telemetry v0.0.0-20240522233618-39ace7a40ae7 // indirect
	golang.org/x/tools v0.29.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
)

tool (
	github.com/magefile/mage
	golang.org/x/vuln/cmd/govulncheck
)
