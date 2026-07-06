-- name: GetAdminByEmail :one
SELECT id, email, password_hash FROM admins WHERE email = $1 LIMIT 1;

-- name: CreateAdminSession :exec
INSERT INTO admin_sessions (admin_id, token_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetAdminSession :one
SELECT id, admin_id, token_id, expires_at, revoked_at
FROM admin_sessions
WHERE token_id = $1 LIMIT 1;

-- name: RevokeAdminSession :exec
UPDATE admin_sessions
SET revoked_at = NOW(), revoked_by = $2
WHERE token_id = $1;

-- name: ListAdminSessions :many
SELECT id, token_id, expires_at, revoked_at, created_at
FROM admin_sessions
WHERE admin_id = $1
ORDER BY created_at DESC;

-- name: RevokeAdminSessionByID :exec
UPDATE admin_sessions
SET revoked_at = NOW(), revoked_by = $2
WHERE id = $1;

-- name: GetAdminProfile :one
SELECT id, email, created_at FROM admins where id = $1;