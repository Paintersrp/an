package config

type ConfigInitError struct {
	msg string
}

func (e *ConfigInitError) Error() string {
	return e.msg
}
