package common

import (
	"database/sql"
	"errors"

	"github.com/mdobak/go-xerrors"

	"github.com/lib/pq"
)

var (
	ErrNoDBConfiguration   = xerrors.New("no DB configuration found")
	ErrFailedToConvertData = xerrors.New("failed to convert data")
	ErrUserIDAlreadyExists = xerrors.New("user id already exists")
	Err                    = NewDefaultErrorMapping()
)

func IsErrAConstraintViolation(err error, constraintType, constraintName string) bool {
	original := errors.Unwrap(errors.Unwrap(err))
	if err, ok := original.(*pq.Error); ok {
		if err.Code.Name() == constraintType && err.Constraint == constraintName {
			return true
		}
	}
	return false
}

type ErrorMap map[string]error

func (e ErrorMap) Handle(name string, err error) error {
	if nerr, ok := e[name]; ok {
		return nerr
	}
	return err
}

func (e ErrorMap) Add(other ErrorMap) {
	for key, value := range other {
		e[key] = value
	}
}

type ErrorMapping struct {
	UniqueViolations     ErrorMap
	ForeignKeyViolations ErrorMap
}

func NewErrorMapping() *ErrorMapping {
	return &ErrorMapping{
		UniqueViolations:     make(ErrorMap),
		ForeignKeyViolations: make(ErrorMap),
	}
}

func (e *ErrorMapping) HandleIgnoreNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return e.Handle(err)
}

func (e *ErrorMapping) Handle(err error) error {
	if err == nil {
		return nil
	}

	original := errors.Unwrap(errors.Unwrap(err))

	var pqerr *pq.Error
	ok := errors.As(original, &pqerr)
	if !ok {
		return err
	}
	switch pqerr.Code.Name() {
	case "unique_violation":
		return e.UniqueViolations.Handle(pqerr.Constraint, err)
	case "foreign_key_violation":
		return e.ForeignKeyViolations.Handle(pqerr.Constraint, err)
	default:
		return err
	}
}

func NewDefaultErrorMapping() *ErrorMapping {
	result := NewErrorMapping()

	result.UniqueViolations.Add(ErrorMap{
		"users_pkey": ErrUserIDAlreadyExists,
	})
	result.ForeignKeyViolations.Add(ErrorMap{})

	return result
}
