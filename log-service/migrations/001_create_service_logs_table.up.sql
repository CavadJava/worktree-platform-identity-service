CREATE TABLE IF NOT EXISTS service_logs (
    id UUID PRIMARY KEY,
    service VARCHAR(100) NOT NULL,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    method VARCHAR(10),
    path VARCHAR(500),
    status INT,
    duration_ms BIGINT,
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_service_logs_service_created ON service_logs (service, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_logs_level ON service_logs (level);
CREATE INDEX IF NOT EXISTS idx_service_logs_created ON service_logs (created_at DESC);
