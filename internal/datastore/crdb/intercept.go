package crdb

//
// import (
// 	"context"
//
// 	"github.com/jackc/pgx/v5"
// 	"github.com/jackc/pgx/v5/pgconn"
// 	"github.com/jackc/pgx/v5/pgxpool"
// )
//
// // interceptingPoolConnReader implements common.DBReader, a subset of the
// // pgxpool.Pool interface, but takes control of connection acquisition instead
// // of delegating it to the pool. This allows the configured executeTxRetryFunc
// // to control resetting the connection on error, even for implicit transactions.
// type interceptingPoolConnReader struct {
// 	pool    *pgxpool.Pool
// 	execute executeTxRetryFunc
// }
//
// // Exec is a reimplementation of pgxpool.Pool.Exec that allows resetting the
// // connection on error via the configured executeTxRetryFunc.
// func (i interceptingPoolConnReader) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
// 	var tag pgconn.CommandTag
// 	err := i.execute(ctx, func(ctx context.Context) error {
// 		conn, err := i.pool.Acquire(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		defer conn.Release()
// 		ctx = ctxPgxConn.WithValue(ctx, conn.Conn())
// 		tag, err = conn.Exec(ctx, sql, arguments...)
// 		return err
// 	})
//
// 	return tag, err
// }
//
// // Query is a reimplementation of pgxpool.Pool.Query that allows resetting the
// // connection on error via the configured executeTxRetryFunc.
// func (i interceptingPoolConnReader) Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error) {
// 	var rows pgx.Rows
// 	err := i.execute(ctx, func(ctx context.Context) error {
// 		conn, err := i.pool.Acquire(ctx)
// 		if err != nil {
// 			rows = errRows{err: err}
// 			return err
// 		}
// 		// conn is not released here, the caller releases it with rows.Close()
// 		ctx = ctxPoolConn.WithValue(ctx, conn)
// 		ctx = ctxPgxConn.WithValue(ctx, conn.Conn())
// 		rows, err = conn.Query(ctx, sql, optionsAndArgs...)
// 		rows = &poolRows{r: rows, err: err, c: conn}
// 		return err
// 	})
//
// 	return rows, err
// }
//
// // QueryRow is a reimplementation of pgxpool.Pool.QueryRow that allows resetting
// // the connection on error via the configured executeTxRetryFunc.
// func (i interceptingPoolConnReader) QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row {
// 	var row pgx.Row
// 	err := i.execute(ctx, func(ctx context.Context) error {
// 		conn, err := i.pool.Acquire(ctx)
// 		if err != nil {
// 			row = errRow{err: err}
// 			return err
// 		}
// 		defer conn.Release()
// 		ctx = ctxPgxConn.WithValue(ctx, conn.Conn())
// 		row = conn.QueryRow(ctx, sql, optionsAndArgs...)
// 		row = &poolRow{r: row, c: conn}
// 		return err
// 	})
// 	if err != nil {
// 		return errRow{err: err}
// 	}
// 	return row
// }
