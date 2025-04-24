package api

const (
	PathPing           = "/"
	PathSummary        = "/summary"
	PathChildParamName = "name"
	PathChild          = "/child"
	PathOneChild       = PathChild + "/{" + PathChildParamName + "}"
	PathStartChild     = PathOneChild + "/start"
	PathStopChild      = PathOneChild + "/stop"
)
