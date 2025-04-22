package api

type Child struct {
	Name string `json:"name" validate:"required"`
	Init []Exec `json:"init"`
	Main Exec   `json:"main" validate:"required"`
}

type Exec struct {
	Cmd  string            `json:"cmd" validate:"required"`
	Args []string          `json:"args"`
	Cwd  string            `json:"cwd,omitzero"`
	Env  map[string]string `json:"env"`
}

type ExecState string

const (
	ExecNotStarted ExecState = "not-started"
	ExecRunning    ExecState = "running"
	ExecEnded      ExecState = "ended"
)

type ExecStatus struct {
	State    ExecState `json:"state"`
	StartErr string    `json:"startErr"`
	Pid      int       `json:"pid,omitzero"`
	ExitCode int       `json:"exitCode"`
}

type ChildStatus struct {
	Init []ExecStatus
	Main ExecStatus
}

type ChildState string

const (
	ChildStopped     ChildState = "stopped"
	ChildInitRunning ChildState = "init-running"
	ChildInitError   ChildState = "init-error"
	ChildRunning     ChildState = "running"
	ChildError       ChildState = "error"
)

type ChildWithStatus struct {
	Child
	State  ChildState  `json:"state"`
	Status ChildStatus `json:"status"`
}

type ChildSummary struct {
	Name  string     `json:"name"`
	State ChildState `json:"state"`
	Pid   int        `json:"pid,omitzero"`
}
