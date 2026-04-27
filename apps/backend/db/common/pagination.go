package common

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultLimit int = 50
	maxLimit     int = 1000
)

func PaginationToBobMods(p pagination.LimitOffsetPagination) []bob.Mod[*dialect.SelectQuery] {
	limit := p.Limit
	offset := p.Offset

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}

	return []bob.Mod[*dialect.SelectQuery]{
		sm.Limit(limit),
		sm.Offset(offset),
	}
}
