CREATE TABLE license_requests (
    id UUID PRIMARY KEY,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requester_name VARCHAR(150) NOT NULL,
    software_product_id UUID NOT NULL REFERENCES software_products(id) ON DELETE RESTRICT,
    software_product_name VARCHAR(150) NOT NULL,
    priority VARCHAR(20) NOT NULL
        CHECK (priority IN ('normal', 'high', 'urgent')),
    reason TEXT NOT NULL CHECK (BTRIM(reason) <> ''),
    status VARCHAR(20) NOT NULL
        CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    selected_license_id UUID REFERENCES licenses(id) ON DELETE RESTRICT,
    selected_license_name VARCHAR(150),
    assignment_id UUID REFERENCES license_assignments(id) ON DELETE RESTRICT,
    reviewed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_by_name VARCHAR(150),
    decision_reason VARCHAR(30)
        CHECK (
            decision_reason IS NULL
            OR decision_reason IN ('out_of_stock', 'not_approved', 'other')
        ),
    response_note TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    reviewed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT ck_license_requests_state CHECK (
        (
            status = 'pending'
            AND selected_license_id IS NULL
            AND selected_license_name IS NULL
            AND assignment_id IS NULL
            AND reviewed_by IS NULL
            AND reviewed_by_name IS NULL
            AND decision_reason IS NULL
            AND response_note IS NULL
            AND reviewed_at IS NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'approved'
            AND selected_license_id IS NOT NULL
            AND BTRIM(COALESCE(selected_license_name, '')) <> ''
            AND assignment_id IS NOT NULL
            AND reviewed_by IS NOT NULL
            AND BTRIM(COALESCE(reviewed_by_name, '')) <> ''
            AND decision_reason IS NULL
            AND reviewed_at IS NOT NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'rejected'
            AND selected_license_id IS NULL
            AND selected_license_name IS NULL
            AND assignment_id IS NULL
            AND reviewed_by IS NOT NULL
            AND BTRIM(COALESCE(reviewed_by_name, '')) <> ''
            AND decision_reason IS NOT NULL
            AND BTRIM(COALESCE(response_note, '')) <> ''
            AND reviewed_at IS NOT NULL
            AND cancelled_at IS NULL
        )
        OR (
            status = 'cancelled'
            AND selected_license_id IS NULL
            AND selected_license_name IS NULL
            AND assignment_id IS NULL
            AND reviewed_by IS NULL
            AND reviewed_by_name IS NULL
            AND decision_reason IS NULL
            AND response_note IS NULL
            AND reviewed_at IS NULL
            AND cancelled_at IS NOT NULL
        )
    )
);

CREATE UNIQUE INDEX uq_pending_license_request
    ON license_requests (requester_id, software_product_id)
    WHERE status = 'pending';

CREATE INDEX idx_license_requests_requester_created
    ON license_requests (requester_id, created_at DESC);

CREATE INDEX idx_license_requests_status_created
    ON license_requests (status, created_at DESC);

CREATE INDEX idx_license_requests_priority
    ON license_requests (priority);

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(80) NOT NULL,
    title VARCHAR(200) NOT NULL CHECK (BTRIM(title) <> ''),
    message TEXT NOT NULL CHECK (BTRIM(message) <> ''),
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    read_at TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;
