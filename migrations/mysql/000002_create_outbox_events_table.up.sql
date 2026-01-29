-- Create outbox_events table
CREATE TABLE IF NOT EXISTS outbox_events (
    id BINARY(16) PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retries INT NOT NULL DEFAULT 0,
    last_error TEXT,
    processed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_outbox_events_status (status),
    INDEX idx_outbox_events_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
