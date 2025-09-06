package state

type State struct {
	Verbose bool
	Quiet   bool
	Debug   bool
	NoColor bool
}

func NewState() *State {
	return &State{
		Verbose: false,
		Quiet:   false,
		Debug:   false,
		NoColor: false,
	}
}
