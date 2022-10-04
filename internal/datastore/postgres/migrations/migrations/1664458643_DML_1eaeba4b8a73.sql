BEGIN;
INSERT INTO alembic_version (version_num) VALUES ('');
INSERT INTO relation_tuple_transaction (timestamp) VALUES (to_timestamp(0));
COMMIT;
