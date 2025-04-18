ALTER TABLE shares
ADD CONSTRAINT unique_file_share UNIQUE (shared_by, shared_for, file_id);
