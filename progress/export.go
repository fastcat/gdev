package progress

import "github.com/jedib0t/go-pretty/v6/progress"

// re-export types we want to expose as-is

type Tracker = progress.Tracker

const (
	PositionLeft  = progress.PositionLeft
	PositionRight = progress.PositionRight
)

var (
	UnitsDefault = progress.UnitsDefault
	UnitsBytes   = progress.UnitsBytes
	// Currency units are unlikely to be used in this context
)
