package service

type Mode int

const (
	ModeDefault  Mode = iota // default
	ModeLocal                // local
	ModeDebug                // debug
	ModeDisabled             // disabled

	// TODO: consider an excluded mode that makes us just pretend the service
	// isn't registered, we neither stop nor start any of its services.
)

//go:generate go tool stringer -type=Mode -linecomment

func (m Mode) Valid() bool {
	return m >= 0 && m < Mode(len(_Mode_index)-1)
}

func ParseMode(value string) (Mode, bool) {
	for i := range len(_Mode_index) - 1 {
		if value == _Mode_name[_Mode_index[i]:_Mode_index[i+1]] {
			return Mode(i), true
		}
	}
	return Mode(0), false
}
