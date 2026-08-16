CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    code VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_departments_name ON departments (LOWER(name));
CREATE UNIQUE INDEX uq_departments_code ON departments (LOWER(code));

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    employee_code VARCHAR(50) NOT NULL UNIQUE,
    role VARCHAR(30) NOT NULL CHECK (role IN ('admin', 'it_manager', 'employee')),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'locked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assigned_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    asset_code VARCHAR(80) NOT NULL,
    serial_number VARCHAR(150),
    name VARCHAR(150) NOT NULL,
    device_type VARCHAR(50) NOT NULL,
    manufacturer VARCHAR(100),
    model VARCHAR(100),
    status VARCHAR(30) NOT NULL DEFAULT 'available'
        CHECK (status IN ('available', 'assigned', 'maintenance', 'retired', 'lost')),
    purchased_at DATE,
    warranty_expires_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_devices_asset_code ON devices (LOWER(asset_code));
CREATE UNIQUE INDEX uq_devices_serial_number
    ON devices (LOWER(serial_number)) WHERE serial_number IS NOT NULL;

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by_hash CHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX idx_refresh_tokens_user_active
    ON refresh_tokens (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE software_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    publisher VARCHAR(150) NOT NULL,
    version VARCHAR(80) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_software_product_identity
    ON software_products (LOWER(name), LOWER(publisher), LOWER(version));

CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    software_product_id UUID NOT NULL REFERENCES software_products(id) ON DELETE RESTRICT,
    name VARCHAR(150) NOT NULL,
    license_type VARCHAR(30) NOT NULL CHECK (license_type IN ('subscription', 'perpetual')),
    assignment_type VARCHAR(30) NOT NULL CHECK (assignment_type IN ('user', 'device', 'mixed')),
    seat_count INTEGER NOT NULL CHECK (seat_count > 0),
    encrypted_key BYTEA,
    key_hint VARCHAR(50),
    vendor VARCHAR(150),
    purchased_at DATE,
    starts_at DATE,
    expires_at DATE,
    cost NUMERIC(14, 2) CHECK (cost IS NULL OR cost >= 0),
    currency CHAR(3),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at IS NULL OR starts_at IS NULL OR expires_at >= starts_at),
    CHECK (license_type <> 'subscription' OR expires_at IS NOT NULL)
);

CREATE TABLE license_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id UUID NOT NULL REFERENCES licenses(id) ON DELETE RESTRICT,
    user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    device_id UUID REFERENCES devices(id) ON DELETE RESTRICT,
    assigned_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    notes TEXT,
    CHECK ((user_id IS NOT NULL)::INTEGER + (device_id IS NOT NULL)::INTEGER = 1),
    CHECK (
        (status = 'active' AND revoked_at IS NULL AND revoked_by IS NULL)
        OR (status = 'revoked' AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_active_license_user
    ON license_assignments (license_id, user_id)
    WHERE status = 'active' AND user_id IS NOT NULL;

CREATE UNIQUE INDEX uq_active_license_device
    ON license_assignments (license_id, device_id)
    WHERE status = 'active' AND device_id IS NOT NULL;

CREATE INDEX idx_active_assignments_license
    ON license_assignments (license_id)
    WHERE status = 'active';

CREATE INDEX idx_licenses_expires_at ON licenses (expires_at);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_created ON audit_logs (actor_id, created_at DESC);
