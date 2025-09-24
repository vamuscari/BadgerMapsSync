package state

// State represents flags passed in. Do not set defaults for these.

type State struct {
	Verbose           bool
	Quiet             bool
	Debug             bool
	NoColor           bool
	NoInput           bool
	ConfigFile        *string
	IsGui             bool
	LogFile           string
	PIDFile           string
	ServerHost        string
	ServerPort        int
	TLSEnabled        bool
	TLSCert           string
	TLSKey            string
	ServerLogRequests bool
}

// NewState creates a new State object with default values
func NewState() *State {
	return &State{
		Verbose:    false,
		Quiet:      false,
		Debug:      false,
		NoColor:    false,
		ConfigFile: new(string),
		NoInput:    false,
	}
}
