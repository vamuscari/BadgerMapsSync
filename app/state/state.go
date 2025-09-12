package state

// State represents flags passed in. Do not set defaults for these.

type State struct {
	Verbose    bool
	Quiet      bool
	Debug      bool
	NoColor    bool
	EnvFile    *string
	ConfigFile *string

	// Server configuration
	ServerHost   string
	ServerPort   int
	TLSEnabled   bool
	TLSCert      string
	TLSKey       string
	WebhookToken string
}

func NewState() *State {
	return &State{
		Verbose:    false,
		Quiet:      false,
		Debug:      false,
		NoColor:    false,
		ConfigFile: new(string),
		EnvFile:    new(string),
	}
}
