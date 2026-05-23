CREATE INDEX idx_audit_logs_client_id_created_at ON audit_logs (client_id, created_at DESC);
