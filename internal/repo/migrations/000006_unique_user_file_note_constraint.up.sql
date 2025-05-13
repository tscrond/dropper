ALTER TABLE notes ADD CONSTRAINT unique_user_file_note UNIQUE (user_id, file_id);
