package fsbackend

// Config defines filesystem specific
// parameters and authetication credentials
type Config struct {
	RootDirectory string `yaml:"rootDirectory"` // filesystem root directory
}
