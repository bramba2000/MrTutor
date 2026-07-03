-- name: GetUserById :one
-- GetUserById retrieves user by id
SELECT
    *
FROM
    users
WHERE
    id = :id
LIMIT
    1;

-- name: GetUserByEmailOrUsername :one
-- GetUserByEmailOrUsername retrieves user by email or username
SELECT
    *
FROM
    users
WHERE
    email = :input
    OR username = :input
LIMIT
    1;

-- name: CreatePrincipal :one
-- CreatePrincipal creates a new user
INSERT INTO
    users (email, username, password, role)
VALUES
    (:email, :username, :password, :role)
RETURNING
    *;

-- name: UpdatePrincipal :one
-- UpdatePrincipal updates user information
UPDATE users
SET
    email = COALESCE(:email, email),
    username = COALESCE(:username, username),
    password = COALESCE(:password, password),
    role = COALESCE(:role, role),
    modified_at = datetime()
WHERE
    id = :id
RETURNING
    *;
