ALTER TABLE clients
  DROP COLUMN IF EXISTS locked_until,
  DROP COLUMN IF EXISTS failed_attempts;
