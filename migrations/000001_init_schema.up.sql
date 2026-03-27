-- Photos table
CREATE TABLE photos (
    photo_id UUID PRIMARY KEY,
    route_id VARCHAR(50) NOT NULL,
    lane_code VARCHAR(10) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    sta_value DOUBLE PRECISION NOT NULL,
    sta_source VARCHAR(20) NOT NULL DEFAULT 'user_provided',
    original_path VARCHAR(500) NOT NULL,
    thumbnail_small VARCHAR(500),
    thumbnail_medium VARCHAR(500),
    thumbnail_large VARCHAR(500),
    file_format VARCHAR(10) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    original_filename VARCHAR(255),
    exif_data JSONB,
    description TEXT,
    tags JSONB DEFAULT '[]',
    upload_token UUID NOT NULL UNIQUE,
    upload_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    uploaded_by VARCHAR(100) NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'processing',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processing_completed_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by VARCHAR(100)
);

-- Pending uploads table
CREATE TABLE pending_uploads (
    upload_token UUID PRIMARY KEY,
    photo_id UUID NOT NULL REFERENCES photos(photo_id),
    api_key_id VARCHAR(100) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    gcs_object_name VARCHAR(500) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
);

-- API keys table
CREATE TABLE api_keys (
    key_id UUID PRIMARY KEY,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    scopes JSONB NOT NULL DEFAULT '["read"]',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Photo audit log table
CREATE TABLE photo_audit_log (
    log_id UUID PRIMARY KEY,
    photo_id UUID REFERENCES photos(photo_id),
    operation VARCHAR(50) NOT NULL,
    api_key_id VARCHAR(100) NOT NULL,
    operated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    details JSONB
);

-- Indexes for photos
CREATE INDEX idx_photos_route_sta ON photos(route_id, sta_value) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_route_lane_sta ON photos(route_id, lane_code, sta_value) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_uploaded_by ON photos(uploaded_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_status ON photos(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_upload_token ON photos(upload_token);

-- Indexes for pending uploads
CREATE INDEX idx_pending_uploads_token ON pending_uploads(upload_token) WHERE status = 'pending';
CREATE INDEX idx_pending_uploads_expires ON pending_uploads(expires_at) WHERE status = 'pending';
CREATE INDEX idx_pending_uploads_api_key ON pending_uploads(api_key_id) WHERE status IN ('pending', 'uploaded');
CREATE INDEX idx_pending_uploads_photo_id ON pending_uploads(photo_id);

-- Indexes for API keys
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Indexes for audit log
CREATE INDEX idx_audit_photo ON photo_audit_log(photo_id, operated_at DESC);
CREATE INDEX idx_audit_api_key ON photo_audit_log(api_key_id, operated_at DESC);
