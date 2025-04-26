ALTER TABLE files 
ADD CONSTRAINT files_unique UNIQUE (owner_google_id, md5_checksum,file_name);
