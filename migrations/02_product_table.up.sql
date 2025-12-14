CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    seller_id UUID NOT NULL,
    images TEXT[] NOT NULL DEFAULT '{}',
    min_price INTEGER NOT NULL,
    current_price INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sold_at TIMESTAMP,
    sold_to UUID,
    CONSTRAINT fk_seller FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_sold_to FOREIGN KEY (sold_to) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_products_seller_id ON products(seller_id);
CREATE INDEX idx_products_sold_to ON products(sold_to);
CREATE INDEX idx_products_created_at ON products(created_at);