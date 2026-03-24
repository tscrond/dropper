-- Prefix all existing flat file_name values with "Main/"
-- Only affects rows that don't already contain a slash (i.e. not yet path-aware)
UPDATE files
SET file_name = 'Main/' || file_name
WHERE file_name NOT LIKE '%/%';
