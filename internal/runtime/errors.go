package runtime

import "fmt"

type OperatorError struct {
	Message    string
	NextAction string
	Cause      error
}

func (e *OperatorError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s; next action: %s", e.Message, e.NextAction)
	}
	return fmt.Sprintf("%s: %s; next action: %s", e.Message, safeErrorText(e.Cause), e.NextAction)
}

func (e *OperatorError) Unwrap() error {
	return e.Cause
}
