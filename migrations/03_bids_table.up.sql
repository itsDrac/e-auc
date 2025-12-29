CREATE TABLE IF NOT EXISTS bids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bid_at TIMESTAMP NOT NULL DEFAULT NOW(),
    product_id UUID NOT NULL,
    user_id UUID NOT NULL,
    price INTEGER NOT NULL,
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    comments TEXT,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bids_product_id ON bids(product_id);
CREATE INDEX IF NOT EXISTS idx_bids_user_id ON bids(user_id);
CREATE INDEX IF NOT EXISTS idx_bids_bid_at ON bids(bid_at);
