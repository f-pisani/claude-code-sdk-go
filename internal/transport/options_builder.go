package transport

// OptionsBuilder interface for building CLI arguments from options
type OptionsBuilder interface {
	BuildCLIArgs() ([]string, error)
}