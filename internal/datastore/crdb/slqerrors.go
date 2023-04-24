package crdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	log "github.com/authzed/spicedb/internal/logging"
)

const (
	// https://www.cockroachlabs.com/docs/stable/common-errors.html#restart-transaction
	crdbRetryErrCode = "40001"
	// https://www.cockroachlabs.com/docs/stable/common-errors.html#result-is-ambiguous
	crdbAmbiguousErrorCode = "40003"
	// https://www.cockroachlabs.com/docs/stable/node-shutdown.html#connection-retry-loop
	crdbServerNotAcceptingClients = "57P01"
	// Error when SqlState is unknown
	crdbUnknownSQLState = "XXUUU"
	// Error message encountered when crdb nodes have large clock skew
	crdbClockSkewMessage = "cannot specify timestamp in the future"
)

// sqlErrorCode attempts to extract the crdb error code from the error state.
func sqlErrorCode(ctx context.Context, err error) string {
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) {
		log.Ctx(ctx).Debug().Err(err).Msg("couldn't determine a sqlstate error code")
		return ""
	}

	return pgerr.SQLState()
}
