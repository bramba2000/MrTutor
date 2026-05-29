CREATE TABLE sessions (
    user_id INTEGER NOT NULL,
    token TEXT PRIMARY KEY NOT NULL,
    absolute_expiry DATETIME NOT NULL,
    idle_expiry DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
