DELETE FROM namespace_config WHERE namespace IN (
    SELECT namespace FROM namespace_config WHERE deleted_transaction = 9223372036854775807 GROUP BY namespace HAVING COUNT(created_transaction) > 1
) AND (namespace, created_transaction) NOT IN (
    SELECT namespace, max(created_transaction) from namespace_config where deleted_transaction = 9223372036854775807 GROUP BY namespace HAVING COUNT(created_transaction) > 1);
