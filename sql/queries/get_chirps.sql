-- name: GetAllChirps :exec 
SELECT body FROM chirps ORDER BY created_at ASC;