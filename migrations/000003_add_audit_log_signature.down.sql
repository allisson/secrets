-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_is_signed;
DROP INDEX IF EXISTS idx_audit_logs_kek_id;

-- Drop foreign key constraint
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_client_id;

-- Drop columns
ALTER TABLE audit_logs DROP COLUMN IF EXISTS is_signed;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS kek_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS signature;
