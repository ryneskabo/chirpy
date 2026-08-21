-- name: UpdateUserEmailAndPassword :one
UPDATE users
SET email = $2,
	hashed_password = $3,
	updated_at = NOW()
WHERE id = $1
RETURNING id, email, updated_at, created_at, is_chirpy_red;
