package common

import (
	"fmt"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type Resolver[T comparable] func(field T) (bob.Expression, error)

func OrderingToBobMods[T comparable](
	orderings ordering.Orderings[T],
	resolve Resolver[T],
) ([]bob.Mod[*dialect.SelectQuery], error) {
	mods := make([]bob.Mod[*dialect.SelectQuery], 0, len(orderings))

	for _, ord := range orderings {
		expr, err := resolve(ord.Field)
		if err != nil {
			return nil, err
		}

		switch ord.Direction {
		case ordering.DirectionAsc:
			mods = append(mods, sm.OrderBy(expr).Asc())
		case ordering.DirectionDesc:
			mods = append(mods, sm.OrderBy(expr).Desc())
		default:
			return nil, fmt.Errorf("unsupported ordering direction: %q", ord.Direction)
		}
	}

	return mods, nil
}
