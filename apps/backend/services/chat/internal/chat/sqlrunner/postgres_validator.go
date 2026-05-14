package sqlrunner

import (
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

type postgresValidator struct{}

func (v postgresValidator) Validate(query string) error {
	tree, err := pgquery.Parse(query)
	if err != nil {
		return invalidSQLf("failed to parse postgres query: %v", err)
	}
	if len(tree.GetStmts()) != 1 {
		return invalidSQL("query must contain exactly one statement")
	}

	stmt := tree.GetStmts()[0].GetStmt()
	selectStmt := stmt.GetSelectStmt()
	if selectStmt == nil {
		return invalidSQL("query must be a read-only select")
	}

	if err := validatePostgresSelect(selectStmt); err != nil {
		return err
	}

	return nil
}

func validatePostgresSelect(stmt *pgquery.SelectStmt) error {
	if stmt == nil {
		return invalidSQL("query must be a read-only select")
	}
	if stmt.GetIntoClause() != nil {
		return invalidSQL("select into is not allowed")
	}
	if len(stmt.GetLockingClause()) > 0 {
		return invalidSQL("locking select clauses are not allowed")
	}
	if err := validatePostgresWithClause(stmt.GetWithClause()); err != nil {
		return err
	}

	// Set operations keep their branches as nested SelectStmt values, so validate
	// them recursively to catch locks or write CTEs inside UNION/INTERSECT/EXCEPT.
	if err := validatePostgresSelectBranch(stmt.GetLarg()); err != nil {
		return err
	}
	if err := validatePostgresSelectBranch(stmt.GetRarg()); err != nil {
		return err
	}

	return nil
}

func validatePostgresSelectBranch(stmt *pgquery.SelectStmt) error {
	if stmt == nil {
		return nil
	}

	return validatePostgresSelect(stmt)
}

func validatePostgresWithClause(with *pgquery.WithClause) error {
	if with == nil {
		return nil
	}

	for _, cteNode := range with.GetCtes() {
		cte := cteNode.GetCommonTableExpr()
		if cte == nil {
			return invalidSQL("common table expression must be a select")
		}

		cteQuery := cte.GetCtequery()
		selectStmt := cteQuery.GetSelectStmt()
		if selectStmt == nil {
			return invalidSQL("data-modifying common table expressions are not allowed")
		}
		if err := validatePostgresSelect(selectStmt); err != nil {
			return err
		}
	}

	return nil
}
