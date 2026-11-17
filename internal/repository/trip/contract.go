package trip

import "github.com/freyliee/template/internal/infra/postgres/transaction"

type database interface{ transaction.Executor }
