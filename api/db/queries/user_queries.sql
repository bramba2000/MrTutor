-- name: GetUserById :one
-- GetUserById retrieves user by id
SELECT * FROM "users" WHERE "id" = :id LIMIT 1;

-- name: GetUserByEmailOrUsername :one
-- GetUserByEmailOrUsername retrieves user by email or username
SELECT * FROM "users" WHERE "email" = :email OR "username" = :username LIMIT 1;

-- name: CreateUser :one
-- CreateUser creates a new user
INSERT INTO "users" ("email", "username", "password") VALUES (:email, :username, :password) RETURNING *;

-- name: UpdateUser :one
-- UpdateUser updates user information
UPDATE "users" SET "email" = :email, "username" = :username, "password" = :password, "modified_at" = datetime() WHERE "id" = :id RETURNING *;
