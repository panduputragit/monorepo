-- name: GetMemberByEmail :one
SELECT id, email, password_hash FROM members WHERE email = $1 LIMIT 1;

-- name: CreateMemberRefreshToken :exec
INSERT INTO member_refresh_tokens (member_id, token_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetMemberRefreshToken :one
SELECT id, member_id, token_id, expires_at, revoked_at
FROM member_refresh_tokens
WHERE token_id = $1 LIMIT 1;

-- name: RevokeMemberRefreshToken :exec
UPDATE member_refresh_tokens
SET revoked_at = NOW()
WHERE token_id = $1;

-- name: GetMemberProfile :one
SELECT id, email, created_at FROM members where id = $1;
