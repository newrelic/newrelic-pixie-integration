package errors

type code string

const (
	cfgErrCode        code = "cfgErr"
	connectionErrCode      = "connErr"
	exporterErrCode        = "expErr"
)

func (c code) String() string {
	return string(c)
}

// ConfigurationError returns an configuration loading error.
func ConfigurationError(msg string) Error {
	return createError(cfgErrCode, 1).
		withMessage(msg)
}

// ConnectionError returns an establishing connection fails.
func ConnectionError(msg string) Error {
	return createError(connectionErrCode, 1).
		withMessage(msg)
}

// ExporterError returns an sending metrics or spans fail.
func ExporterError(msg string) Error {
	return createError(exporterErrCode, 0).
		withMessage(msg)
}
