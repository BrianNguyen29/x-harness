package cli

const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

type CLIError struct {
	Message string
	Code    int
}

func (e *CLIError) Error() string {
	return e.Message
}

func NewError(message string, code int) *CLIError {
	if code == 0 {
		code = ExitError
	}
	return &CLIError{Message: message, Code: code}
}
