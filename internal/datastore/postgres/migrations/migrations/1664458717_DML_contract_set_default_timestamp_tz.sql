ALTER TABLE relation_tuple_transaction
    ALTER COLUMN timestamp SET DEFAULT (now() AT TIME ZONE 'UTC');
