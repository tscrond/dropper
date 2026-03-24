-- Remove duplicate (owner_google_id, file_name) rows keeping the newest (highest id)
DELETE FROM files
WHERE id NOT IN (
    SELECT MAX(id)
    FROM files
    GROUP BY owner_google_id, file_name
);
