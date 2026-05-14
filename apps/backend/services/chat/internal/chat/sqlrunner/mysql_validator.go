package sqlrunner

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

type mysqlValidator struct {
	parser *sqlparser.Parser
}

func newMySQLValidator() mysqlValidator {
	return mysqlValidator{parser: sqlparser.NewTestParser()}
}

func (v mysqlValidator) Validate(query string) error {
	stmts, err := v.parser.ParseMultipleIgnoreEmpty(query)
	if err != nil {
		return invalidSQLf("failed to parse mysql query: %v", err)
	}
	if len(stmts) != 1 {
		return invalidSQL("query must contain exactly one statement")
	}

	selectStmt, ok := stmts[0].(sqlparser.SelectStatement)
	if !ok {
		return invalidSQL("query must be a read-only select")
	}

	if err := validateMySQLSelect(selectStmt); err != nil {
		return err
	}

	return nil
}

func validateMySQLSelect(stmt sqlparser.SelectStatement) error {
	if stmt.GetLock() != sqlparser.NoLock {
		return invalidSQL("locking select clauses are not allowed")
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		if stmt.Into != nil {
			return invalidSQL("select into is not allowed")
		}
		return validateMySQLWithClause(stmt.With)
	case *sqlparser.Union:
		if stmt.Into != nil {
			return invalidSQL("select into is not allowed")
		}
		if err := validateMySQLWithClause(stmt.With); err != nil {
			return err
		}
		if err := validateMySQLTableStatement(stmt.Left); err != nil {
			return err
		}
		return validateMySQLTableStatement(stmt.Right)
	default:
		return invalidSQL("query must be a read-only select")
	}
}

func validateMySQLTableStatement(stmt sqlparser.TableStatement) error {
	selectStmt, ok := stmt.(sqlparser.SelectStatement)
	if !ok {
		return invalidSQL("query must be a read-only select")
	}

	return validateMySQLSelect(selectStmt)
}

func validateMySQLWithClause(with *sqlparser.With) error {
	if with == nil {
		return nil
	}

	for _, cte := range with.CTEs {
		if err := validateMySQLTableStatement(cte.Subquery); err != nil {
			return err
		}
	}

	return nil
}
