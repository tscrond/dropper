CREATE TABLE notes (
    id SERIAL PRIMARY KEY,
    user_id TEXT REFERENCES users(google_id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    content TEXT NOT NULL
);
