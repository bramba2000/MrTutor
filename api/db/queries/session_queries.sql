-- name: GetSessionById :one
-- GetSessionById retrieve an unexperied session
SELECT
    *
FROM
    sessions
WHERE
    token = :token;

-- name: CreateSession :one
-- CreateSession creates a new session for a user
INSERT INTO
    sessions (user_id, token, absolute_expiry, idle_expiry)
VALUES
    (:user_id, :token, :absolute_expiry, :idle_expiry)
RETURNING
    *;

-- name: DeleteExpiredSessions :exec
-- DeleteExpiredSessions deletes all expired sessions
DELETE FROM sessions
WHERE
    absolute_expiry < datetime('now')
    OR idle_expiry < datetime('now');

-- name: DeleteSession :exec
-- DeleteSession deletes a session by token
DELETE FROM sessions
WHERE
    token = :token;

-- name: UpdateSessionIdleExpiry :exec
-- UpdateSessionIdleExpiry updates the idle expiry of a session
UPDATE sessions
SET idle_expiry = :idle_expiry
WHERE
    token = :token;

-- name: GetPrincipalBySessionId :one
-- GetPrincipalBySessionId retrieves a principal by session token
SELECT
    users.id, users.username, users.email, users.password, users.created_at, users.modified_at, users.role
FROM
    users
JOIN
    sessions ON users.id = sessions.user_id
WHERE
    sessions.token = :token AND
    sessions.absolute_expiry > datetime('now') AND
    sessions.idle_expiry > datetime('now');
