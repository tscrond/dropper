-- USERS table: stores info about users in the system
CREATE TABLE users (
    google_id TEXT PRIMARY KEY,
    user_name TEXT,
    user_email TEXT UNIQUE NOT NULL,
    user_bucket TEXT UNIQUE
);


-- FILES table: represents metadata about each file stored in GCS
CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    owner_google_id TEXT REFERENCES users(google_id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_type TEXT,
    size BIGINT,
    md5_checksum TEXT NOT NULL,
    UNIQUE (owner_google_id, md5_checksum)
);

-- SHARES table: represents shared files between users
CREATE TABLE shares (
    id SERIAL PRIMARY KEY,
    shared_by TEXT REFERENCES users(user_email) ON DELETE CASCADE,
    shared_for TEXT,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT unique_file_share UNIQUE (shared_by, shared_for, file_id)
);

-- Index for fast lookup of who a user has shared files with
CREATE INDEX idx_shares_shared_for ON shares(shared_for);
