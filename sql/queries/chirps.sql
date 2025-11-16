-- name: CreateChirpy :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (gen_random_uuid(), now(), now(), $1, $2) RETURNING *;

-- name: DropChirpsTable :exec
DELETE FROM chirps;

-- name: GetAllChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetChirpsById :one
SELECT * FROM chirps WHERE id = $1;
