-- Revert: strip the "Main/" prefix that was added by 000007
UPDATE files
SET file_name = SUBSTRING(file_name FROM 6)
WHERE file_name LIKE 'Main/%';
