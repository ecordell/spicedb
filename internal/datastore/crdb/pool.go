package crdb

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/authzed/spicedb/internal/logging"
)

var resetHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "crdb_client_resets",
	Help:    "cockroachdb client-side tx reset distribution",
	Buckets: []float64{0, 1, 2, 5, 10, 20, 50},
})

func init() {
	prometheus.MustRegister(resetHistogram)
}

type MaxRetryError struct {
	MaxRetries uint8
	LastErr    error
}

func (e *MaxRetryError) Error() string {
	return strconv.Itoa(int(e.MaxRetries)) + "max retries reached" + ": " + e.LastErr.Error()
}
func (e *MaxRetryError) Unwrap() error { return e.LastErr }

type ResettableError struct {
	Err error
}

func (e *ResettableError) Error() string { return "resettable error" + ": " + e.Err.Error() }
func (e *ResettableError) Unwrap() error { return e.Err }

type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return "retryable error" + ": " + e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

type RetryPool struct {
	pool *pgxpool.Pool

	maxRetries uint8
}

var nodeIDs sync.Map

func NewRetryPool(ctx context.Context, config *pgxpool.Config, maxRetries uint8) (*RetryPool, error) {
	afterConnect := config.AfterConnect
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if afterConnect != nil {
			if err := afterConnect(ctx, conn); err != nil {
				return err
			}
		}
		var nodeID string
		if err := conn.QueryRow(ctx, "SHOW node_id;").Scan(&nodeID); err != nil {
			return err
		}

		nodeIDs.Store(conn, nodeID)

		runtime.SetFinalizer(conn, func(conn *pgx.Conn) {
			nodeIDs.Delete(conn)
		})

		return nil
	}
	afterRelease := config.AfterRelease
	config.AfterRelease = func(conn *pgx.Conn) bool {
		if afterRelease != nil && !afterRelease(conn) {
			return false
		}

		if conn.IsClosed() {
			nodeIDs.Delete(conn)
		}
		return true
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &RetryPool{
		pool:       pool,
		maxRetries: maxRetries,
	}, nil
}

func (p *RetryPool) withRetries(ctx context.Context, fn func(conn *pgxpool.Conn) error) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error acquiring connection from pool: %w", err)
	}
	defer conn.Release()

	var retries uint8
	defer func() {
		resetHistogram.Observe(float64(retries))
	}()

	for retries = uint8(0); retries <= p.maxRetries; retries++ {
		if retries > 0 {
			log.Ctx(ctx).Info().Uint8("attempt", retries).Msg("before attempt")
		}
		err = wrap(ctx, fn(conn))
		if retries > 0 {
			log.Ctx(ctx).Info().Uint8("attempt", retries).Msg("after attempt")
		}
		if err == nil {
			if retries > 0 {
				log.Ctx(ctx).Info().Uint8("retries", retries).Msg("resettable database error succeeded after retry")
			}
			return nil
		}
		var (
			resettable *ResettableError
			retryable  *RetryableError
		)
		if errors.As(err, &resettable) {
			id := safeclose(ctx, conn)
			conn.Release()

			sleepOnErr(ctx, err, retries)

			conn, err = p.acquireFromDifferentNode(ctx, id)
			if err != nil {
				return fmt.Errorf("error acquiring connection from pool after retry %d: %w", retries, err)
			}
			continue
		}
		if errors.As(err, &retryable) {
			sleepOnErr(ctx, err, retries)
			continue
		}
		// no error or error is not resettable or retriable
		return err
	}
	return &MaxRetryError{MaxRetries: p.maxRetries, LastErr: err}
}

func safeclose(ctx context.Context, conn *pgxpool.Conn) string {
	defer func() {
		if err := recover(); err != nil {
			log.Ctx(ctx).Info().Err(fmt.Errorf("%v", err)).Msg("connection already released")
		}
	}()
	if err := conn.Conn().Close(ctx); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("error closing connection after resettable error")
	}
	nodeID, _ := nodeIDs.Load(conn.Conn())
	return nodeID.(string)
}

func sleepOnErr(ctx context.Context, err error, retries uint8) {
	after := retry.BackoffExponentialWithJitter(100*time.Millisecond, 0.5)(uint(retries))
	log.Ctx(ctx).Warn().Err(err).Dur("after", after).Msg("retrying on database error")
	time.Sleep(after)
}

func (p *RetryPool) acquireFromDifferentNode(ctx context.Context, nodeID any) (*pgxpool.Conn, error) {
	log.Ctx(ctx).Info().Str("node_id", nodeID.(string)).Msg("acquiring a connection from a different node")
	for {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		id, ok := nodeIDs.Load(conn.Conn())
		if !ok || id == nodeID {
			conn.Release()
			continue
		}
		log.Ctx(ctx).Info().Str("old node id", nodeID.(string)).Str("new node id", id.(string)).Msg("acquired a connection from a different node")
		return conn, nil
	}
}

// ExecFunc is a reimplementation of pgxpool.Pool.Exec that allows resetting the
// connection on error via the configured executeTxRetryFunc.
func (p *RetryPool) ExecFunc(ctx context.Context, tagFunc func(ctx context.Context, tag pgconn.CommandTag, err error) error, sql string, arguments ...any) error {
	return p.withRetries(ctx, func(conn *pgxpool.Conn) error {
		tag, err := conn.Conn().Exec(ctx, sql, arguments...)
		return tagFunc(ctx, tag, err)
	})
}

// QueryFunc is a reimplementation of pgxpool.Pool.Query that allows resetting the
// connection on error via the configured executeTxRetryFunc.
func (p *RetryPool) QueryFunc(ctx context.Context, rowsFunc func(ctx context.Context, rows pgx.Rows) error, sql string, optionsAndArgs ...any) error {
	return p.withRetries(ctx, func(conn *pgxpool.Conn) error {
		rows, err := conn.Conn().Query(ctx, sql, optionsAndArgs...)
		defer rows.Close()
		if err != nil {
			return err
		}
		return rowsFunc(ctx, rows)
	})
}

// QueryRowFunc is a reimplementation of pgxpool.Pool.QueryRow that allows resetting
// the connection on error via the configured executeTxRetryFunc.
func (p *RetryPool) QueryRowFunc(ctx context.Context, rowFunc func(ctx context.Context, row pgx.Row) error, sql string, optionsAndArgs ...any) error {
	return p.withRetries(ctx, func(conn *pgxpool.Conn) error {
		return rowFunc(ctx, conn.Conn().QueryRow(ctx, sql, optionsAndArgs...))
	})
}

func (p *RetryPool) BeginFunc(ctx context.Context, txFunc func(pgx.Tx) error) error {
	return p.withRetries(ctx, func(conn *pgxpool.Conn) error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		return beginFuncExec(ctx, tx, txFunc)
	})
}

func (p *RetryPool) BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, txFunc func(pgx.Tx) error) error {
	return p.withRetries(ctx, func(conn *pgxpool.Conn) error {
		tx, err := conn.BeginTx(ctx, txOptions)
		if err != nil {
			return err
		}
		return beginFuncExec(ctx, tx, txFunc)
	})
}

func (p *RetryPool) Close() {
	p.pool.Close()
}

func (p *RetryPool) Stat() *pgxpool.Stat {
	return p.pool.Stat()
}

// this fn is copied from pgx
func beginFuncExec(ctx context.Context, tx pgx.Tx, fn func(pgx.Tx) error) (err error) {
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	fErr := fn(tx)
	if fErr != nil {
		_ = tx.Rollback(ctx) // ignore rollback error as there is already an error to return
		return fErr
	}

	return tx.Commit(ctx)
}

func wrap(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// if context is errored or timed out, return immediately
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return err
	}

	// detect when an error is likely due to a node taken out of service
	if strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "unexpected EOF") ||
		strings.Contains(err.Error(), "conn closed") ||
		strings.Contains(err.Error(), "connection reset by peer") {
		return &ResettableError{Err: err}
	}

	sqlState := sqlErrorCode(ctx, err)
	// Ambiguous result error includes connection closed errors
	// https://www.cockroachlabs.com/docs/stable/common-errors.html#result-is-ambiguous
	if sqlState == crdbAmbiguousErrorCode ||
		// Reset on node draining
		sqlState == crdbServerNotAcceptingClients {
		return &ResettableError{Err: err}
	}

	// Retryable errors: the transaction should be retried but no new connection
	// is needed.
	if sqlState == crdbRetryErrCode ||
		// Error encountered when crdb nodes have large clock skew
		(sqlState == crdbUnknownSQLState && strings.Contains(err.Error(), crdbClockSkewMessage)) {
		return &RetryableError{Err: err}
	}
	return err
}
