CREATE TABLE IF NOT EXISTS product_items (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    price NUMERIC(12,2) NOT NULL DEFAULT 0,
    stock INTEGER NOT NULL DEFAULT 0,
    is_discounted BOOLEAN NOT NULL DEFAULT false,
    discount_price NUMERIC(12,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_items_product_id ON product_items (product_id);
