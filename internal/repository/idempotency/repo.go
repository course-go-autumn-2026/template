package idempotency

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/toooriii/tori/internal/infra/postgres/querier"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Repo struct {
	q querier.Provider
}

func New(q querier.Provider) *Repo {
	return &Repo{q: q}
}
