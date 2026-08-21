-- name: DeleteAChirp :exec
DELETE FROM chirps
WHERE id = $1;
