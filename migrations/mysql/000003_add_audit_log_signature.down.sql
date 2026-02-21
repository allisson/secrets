-- Drop indexes
DROP INDEX idx_audit_logs_is_signed ON audit_logs;
DROP INDEX idx_audit_logs_kek_id ON audit_logs;

-- Drop foreign key constraints
ALTER TABLE audit_logs DROP FOREIGN KEY fk_audit_logs_client_id;
ALTER TABLE audit_logs DROP FOREIGN KEY fk_audit_logs_kek_id;

-- Drop columns
ALTER TABLE audit_logs DROP COLUMN is_signed;
ALTER TABLE audit_logs DROP COLUMN kek_id;
ALTER TABLE audit_logs DROP COLUMN signature;
