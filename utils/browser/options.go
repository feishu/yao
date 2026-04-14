package browser

import (
	"fmt"
)

const (
	outputBase64 = "base64"
	outputFile   = "file"
)

type validationError struct {
	message string
}

func (err validationError) Error() string {
	return err.message
}

func newValidationError(format string, args ...interface{}) error {
	return validationError{message: fmt.Sprintf(format, args...)}
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(validationError)
	return ok
}

func normalizeOptions(options Options) (Options, error) {
	if options.Output == "" {
		options.Output = outputBase64
	}

	switch options.Output {
	case outputBase64, outputFile:
	default:
		return options, newValidationError("invalid output type: %s", options.Output)
	}

	if options.Output == outputFile && options.Filename == "" {
		return options, newValidationError("filename is required when output is file")
	}

	if options.Width <= 0 {
		options.Width = 1280
	}

	if options.Height <= 0 {
		options.Height = 720
	}

	if options.Timeout <= 0 {
		options.Timeout = 30000
	}

	return options, nil
}
