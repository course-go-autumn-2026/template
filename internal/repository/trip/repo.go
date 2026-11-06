package trip

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/toooriii/tori/internal/infra/postgres/querier"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type TripRepository struct {
	q querier.Provider
}

func New(q querier.Provider) *TripRepository {
	return &TripRepository{q: q}
}
