-- name: CreateBid :exec
INSERT INTO bids (
    product_id,
    user_id,
    price,
    comments
) VALUES (
    $1, $2, $3, $4
);

-- name: GetBidsByProductID :many
SELECT * FROM bids
WHERE product_id = $1
ORDER BY bid_at DESC;

-- name: GetBidsByUserID :many
SELECT * FROM bids
WHERE user_id = $1
ORDER BY bid_at DESC;

-- name: GetValidBidsByProductID :many
SELECT * FROM bids
WHERE product_id = $1 AND is_valid = true
ORDER BY bid_at DESC;

-- name: InvalidateBid :exec
UPDATE bids
SET is_valid = false
WHERE id = $1;

-- name: GetLatestBidForProduct :one
SELECT * FROM bids
WHERE product_id = $1 AND is_valid = true
ORDER BY bid_at DESC
LIMIT 1;

-- name: CountBidsByProduct :one
SELECT COUNT(*) FROM bids
WHERE product_id = $1 AND is_valid = true;

-- name: DeleteBid :exec
DELETE FROM bids
WHERE id = $1;
