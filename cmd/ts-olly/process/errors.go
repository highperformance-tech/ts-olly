package process

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrConfigDirNotFound  Error = "config directory not found"
	ErrConfigFileNotFound Error = "config file not found"
	ErrInvalidConfigFile  Error = "invalid config file"
)
