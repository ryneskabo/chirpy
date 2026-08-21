-- name: GetUserByEmail :one
SELECT
	id,
	created_at,
	updated_at,
	email,
	is_chirpy_red
FROM users
WHERE email = $1;
