ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_licenses_archived_at
    ON licenses (archived_at)
    WHERE archived_at IS NOT NULL;
