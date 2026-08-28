CREATE TABLE maintenance_requests (
    id UUID PRIMARY KEY,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requester_name VARCHAR(150) NOT NULL CHECK (BTRIM(requester_name) <> ''),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    device_asset_code VARCHAR(80) NOT NULL CHECK (BTRIM(device_asset_code) <> ''),
    device_serial_number VARCHAR(150),
    device_name VARCHAR(150) NOT NULL CHECK (BTRIM(device_name) <> ''),
    device_type VARCHAR(80) NOT NULL CHECK (BTRIM(device_type) <> ''),
    device_manufacturer VARCHAR(150),
    device_model VARCHAR(150),
    device_purchased_at DATE,
    device_warranty_expires_at DATE,
    category VARCHAR(30) NOT NULL
        CHECK (category IN ('hardware', 'software', 'network', 'accessory', 'other')),
    priority VARCHAR(20) NOT NULL
        CHECK (priority IN ('normal', 'high', 'urgent')),
    title VARCHAR(200) NOT NULL CHECK (BTRIM(title) <> ''),
    description TEXT NOT NULL CHECK (BTRIM(description) <> ''),
    status VARCHAR(20) NOT NULL
        CHECK (status IN ('pending', 'in_progress', 'completed', 'rejected', 'cancelled')),
    assigned_to UUID REFERENCES users(id) ON DELETE RESTRICT,
    assigned_to_name VARCHAR(150),
    last_actor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    last_actor_name VARCHAR(150) NOT NULL CHECK (BTRIM(last_actor_name) <> ''),
    response_note TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT ck_maintenance_requests_state CHECK (
        (
            status = 'pending'
            AND assigned_to IS NULL
            AND assigned_to_name IS NULL
            AND response_note IS NULL
            AND accepted_at IS NULL
            AND completed_at IS NULL
            AND rejected_at IS NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'in_progress'
            AND assigned_to IS NOT NULL
            AND BTRIM(COALESCE(assigned_to_name, '')) <> ''
            AND response_note IS NULL
            AND accepted_at IS NOT NULL
            AND completed_at IS NULL
            AND rejected_at IS NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'completed'
            AND assigned_to IS NOT NULL
            AND BTRIM(COALESCE(assigned_to_name, '')) <> ''
            AND BTRIM(COALESCE(response_note, '')) <> ''
            AND accepted_at IS NOT NULL
            AND completed_at IS NOT NULL
            AND rejected_at IS NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'rejected'
            AND BTRIM(COALESCE(response_note, '')) <> ''
            AND completed_at IS NULL
            AND rejected_at IS NOT NULL
            AND cancelled_at IS NULL
            AND (
                (assigned_to IS NULL AND assigned_to_name IS NULL AND accepted_at IS NULL)
                OR (
                    assigned_to IS NOT NULL
                    AND BTRIM(COALESCE(assigned_to_name, '')) <> ''
                    AND accepted_at IS NOT NULL
                )
            )
        )
        OR (
            status = 'cancelled'
            AND assigned_to IS NULL
            AND assigned_to_name IS NULL
            AND response_note IS NULL
            AND accepted_at IS NULL
            AND completed_at IS NULL
            AND rejected_at IS NULL
            AND cancelled_at IS NOT NULL
        )
    )
);

CREATE UNIQUE INDEX uq_open_maintenance_request
    ON maintenance_requests (device_id)
    WHERE status IN ('pending', 'in_progress');

CREATE INDEX idx_maintenance_requests_requester_created
    ON maintenance_requests (requester_id, created_at DESC);

CREATE INDEX idx_maintenance_requests_status_created
    ON maintenance_requests (status, created_at DESC);

CREATE INDEX idx_maintenance_requests_priority
    ON maintenance_requests (priority);

CREATE INDEX idx_maintenance_requests_category
    ON maintenance_requests (category);
