package fsbackend

// Config defines filesystem specific
// parameters and authetication credentials
type Config struct {
	RootDirectory string `yaml:"root_directory"` // filesystem root directory
}
