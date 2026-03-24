-- Replace the old checksum-inclusive unique constraint with a path-only constraint.
-- This enforces that a user cannot have two files at the same canonical path.
ALTER TABLE files DROP CONSTRAINT IF EXISTS files_unique;

ALTER TABLE files
    ADD CONSTRAINT files_unique_path UNIQUE (owner_google_id, file_name);
