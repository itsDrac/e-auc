-- name: AddProduct :one
INSERT INTO products (
    title,
    description,
    seller_id,
    images,
    min_price,
    current_price
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetProductImages :one
SELECT images FROM products
WHERE id = $1
LIMIT 1;

-- name: UpdateProductImages :one
UPDATE products
SET images = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1
LIMIT 1;

-- name: GetProductsBySellerID :many
SELECT * FROM products
WHERE seller_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkProductAsSold :one
UPDATE products
SET sold_at = NOW(), sold_to = $2, current_price = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProductCurrentPrice :exec
UPDATE products
SET current_price = $2, updated_at = NOW()
WHERE id = $1;