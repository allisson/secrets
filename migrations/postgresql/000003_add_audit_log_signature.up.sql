-- Add cryptographic signature columns to audit_logs for tamper detection (PCI DSS Requirement 10.2.2)
ALTER TABLE audit_logs 
ADD COLUMN signature BYTEA,
ADD COLUMN kek_id UUID REFERENCES keks(id) ON DELETE RESTRICT,
ADD COLUMN is_signed BOOLEAN NOT NULL DEFAULT FALSE;

-- Add foreign key constraint for client_id to prevent deletion of clients with audit logs
ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_client_id FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE RESTRICT;

-- Create indexes for efficient queries
CREATE INDEX idx_audit_logs_kek_id ON audit_logs(kek_id);
CREATE INDEX idx_audit_logs_is_signed ON audit_logs(is_signed);

-- Mark existing logs as legacy (unsigned)
UPDATE audit_logs SET is_signed = FALSE WHERE signature IS NULL;
