-- name: GetUserByEmail :one
SELECT
	id,
	created_at,
	updated_at,
	email
FROM users
WHERE email = $1;
