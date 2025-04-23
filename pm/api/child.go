package api

import "time"

type Child struct {
	Name        string       `json:"name" validate:"required"`
	Init        []Exec       `json:"init"`
	Main        Exec         `json:"main" validate:"required"`
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
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
	ExecStopping   ExecState = "stopping"
	ExecEnded      ExecState = "ended"
)

type ExecStatus struct {
	State    ExecState `json:"state"`
	StartErr string    `json:"startErr"`
	Pid      int       `json:"pid,omitzero"`
	ExitCode int       `json:"exitCode"`
}

type HealthStatus struct {
	Healthy       bool       `json:"healthy"`
	LastHealthy   *time.Time `json:"lastHealthy,omitempty"`
	LastUnhealthy *time.Time `json:"lastUnhealthy,omitempty"`
}

type ChildStatus struct {
	State  ChildState   `json:"state"`
	Init   []ExecStatus `json:"init"`
	Main   ExecStatus   `json:"main"`
	Health HealthStatus `json:"health"`
}

type ChildState string

const (
	ChildStopped     ChildState = "stopped"
	ChildInitRunning ChildState = "init-running"
	ChildInitError   ChildState = "init-error"
	ChildRunning     ChildState = "running"
	ChildStopping    ChildState = "stopping"
	ChildError       ChildState = "error"
)

type ChildWithStatus struct {
	Child
	Status ChildStatus `json:"status"`
}

type ChildSummary struct {
	Name  string     `json:"name"`
	State ChildState `json:"state"`
	Pid   int        `json:"pid,omitzero"`
}

type HealthCheck struct {
	Http           *HttpHealthCheck `json:"http,omitempty" validate:"required"`
	TimeoutSeconds int              `json:"timeout" validate:"required,gt=0"`
}

// FUTURE: add more health check types, validate via `required_without_all=<others>,excluded_with=<others>`

type HttpHealthCheck struct {
	Port int    `json:"port" validate:"required,gt=0,lte=65535"`
	Path string `json:"path" validate:"required"`
}
