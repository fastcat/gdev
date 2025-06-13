package service

type Mode int

const (
	ModeDefault Mode = iota // default
	ModeLocal               // local
	ModeDebug               // debug
)

//go:generate go tool stringer -type=Mode -linecomment

func (m Mode) Valid() bool {
	return m >= 0 && m < Mode(len(_Mode_index)-1)
}
