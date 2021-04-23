package errors

type code string

const (
	cfgErrCode code = "configErr"
)

func (c code) String() string {
	return string(c)
}

// ConfigurationError returns an configuration loading error.
func ConfigurationError(msg string) Error {
	return createError(cfgErrCode, 1).
		withMessage(msg)
}