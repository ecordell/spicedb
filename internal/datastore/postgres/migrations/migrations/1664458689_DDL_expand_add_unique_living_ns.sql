ALTER TABLE namespace_config
ADD CONSTRAINT uq_namespace_living UNIQUE (namespace, deleted_transaction);
