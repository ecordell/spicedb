package migrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

const insertUniqueID = `INSERT INTO metadata (unique_id) VALUES ($1);`

func setUniqueID(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, insertUniqueID, uuid.NewString())
	return err
}
