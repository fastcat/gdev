module fastcat.org/go/gdev/magefiles

go 1.25

require (
	github.com/goccy/go-yaml v1.19.2
	github.com/magefile/mage v1.15.0
	golang.org/x/mod v0.33.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260116145544-c6413dc483f5 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/tools/go/expect v0.1.1-deprecated // indirect
	golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated // indirect
	golang.org/x/vuln v1.1.4 // indirect
)

tool (
	github.com/magefile/mage
	golang.org/x/tools/cmd/stringer
	golang.org/x/vuln/cmd/govulncheck
)
