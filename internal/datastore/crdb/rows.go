package crdb

// This file (and only this file) is duplicated from
// github.com/jackc/pgx/v5 under the MIT License.

// Copyright (c) 2013-2021 Jack Christensen
//
// MIT License
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errRow is copied from github.com/jackc/pgx/v5@v5.3.1/pgxpool/rows.go
// and is used in return values of interceptingPoolConnReader to satisfy the
// pool interface (the same as it is used by pgxpool.Pool)
type errRow struct {
	err error
}

func (e errRow) Scan(...any) error { return e.err }

// errRows is copied from github.com/jackc/pgx/v5@v5.3.1/pgxpool/rows.go
// and is used in return values of interceptingPoolConnReader to satisfy the
// pool interface (the same as it is used by pgxpool.Pool)
type errRows struct {
	err error
}

func (errRows) Close()                                       {}
func (e errRows) Err() error                                 { return e.err }
func (errRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (errRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (errRows) Next() bool                                   { return false }
func (e errRows) Scan(...any) error                          { return e.err }
func (e errRows) Values() ([]any, error)                     { return nil, e.err }
func (e errRows) RawValues() [][]byte                        { return nil }
func (e errRows) Conn() *pgx.Conn                            { return nil }

// poolRows is copied from github.com/jackc/pgx/v5@v5.3.1/pgxpool/rows.go
// and is used in return values of interceptingPoolConnReader to satisfy the
// pool interface (the same as it is used by pgxpool.Pool)
type poolRows struct {
	r   pgx.Rows
	c   *pgxpool.Conn
	err error
}

func (rows *poolRows) Close() {
	rows.r.Close()
	if rows.c != nil {
		rows.c.Release()
		rows.c = nil
	}
}

func (rows *poolRows) Err() error {
	if rows.err != nil {
		return rows.err
	}
	return rows.r.Err()
}

func (rows *poolRows) CommandTag() pgconn.CommandTag {
	return rows.r.CommandTag()
}

func (rows *poolRows) FieldDescriptions() []pgconn.FieldDescription {
	return rows.r.FieldDescriptions()
}

func (rows *poolRows) Next() bool {
	if rows.err != nil {
		return false
	}

	n := rows.r.Next()
	if !n {
		rows.Close()
	}
	return n
}

func (rows *poolRows) Scan(dest ...any) error {
	err := rows.r.Scan(dest...)
	if err != nil {
		rows.Close()
	}
	return err
}

func (rows *poolRows) Values() ([]any, error) {
	values, err := rows.r.Values()
	if err != nil {
		rows.Close()
	}
	return values, err
}

func (rows *poolRows) RawValues() [][]byte {
	return rows.r.RawValues()
}

func (rows *poolRows) Conn() *pgx.Conn {
	return rows.r.Conn()
}

type poolRow struct {
	r   pgx.Row
	c   *pgxpool.Conn
	err error
}

func (row *poolRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}

	err := row.r.Scan(dest...)
	if row.c != nil {
		row.c.Release()
	}
	return err
}
