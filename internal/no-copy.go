package internal

type NoCopy struct{}

func (NoCopy) Lock()   {}
func (NoCopy) Unlock() {}
