package sqlrunner

import "errors"

type ErrorKind string

const (
	ErrorKindValidation     ErrorKind = "validation"
	ErrorKindQueryExecution ErrorKind = "query_execution"
	ErrorKindRuntime        ErrorKind = "runtime"
)

type Error struct {
	Kind               ErrorKind
	CorrectionEligible bool
	Err                error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsCorrectionEligible(err error) bool {
	var runnerErr *Error
	if errors.As(err, &runnerErr) {
		return runnerErr.CorrectionEligible
	}
	return false
}

func Kind(err error) (ErrorKind, bool) {
	var runnerErr *Error
	if errors.As(err, &runnerErr) {
		return runnerErr.Kind, true
	}
	return "", false
}

func validationError(err error) error {
	return &Error{
		Kind:               ErrorKindValidation,
		CorrectionEligible: true,
		Err:                err,
	}
}

func queryExecutionError(err error) error {
	return &Error{
		Kind:               ErrorKindQueryExecution,
		CorrectionEligible: true,
		Err:                err,
	}
}

func runtimeError(err error) error {
	return &Error{
		Kind:               ErrorKindRuntime,
		CorrectionEligible: false,
		Err:                err,
	}
}
