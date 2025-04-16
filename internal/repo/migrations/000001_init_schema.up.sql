-- USERS table: stores info about users in the system
CREATE TABLE users (
    google_id TEXT PRIMARY KEY,
    user_name TEXT,
    user_email TEXT UNIQUE NOT NULL
);

-- FILES table: represents metadata about each file stored in GCS
CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    owner_google_id TEXT REFERENCES users(google_id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_type TEXT,
    size BIGINT,
    md5_checksum TEXT NOT NULL UNIQUE
);

-- SHARES table: represents shared files between users
CREATE TABLE shares (
    id SERIAL PRIMARY KEY,
    shared_by TEXT REFERENCES users(google_id) ON DELETE CASCADE,
    shared_for TEXT REFERENCES users(google_id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    sharing_link TEXT,  -- can be NULL and generated on demand
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Index for fast lookup of who a user has shared files with
CREATE INDEX idx_shares_shared_for ON shares(shared_for);
