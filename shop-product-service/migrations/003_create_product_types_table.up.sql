ALTER TABLE products ADD COLUMN IF NOT EXISTS product_type_id UUID;

CREATE TABLE IF NOT EXISTS product_types (
    id UUID PRIMARY KEY,
    shop_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_types_shop_id ON product_types (shop_id);

CREATE TABLE IF NOT EXISTS product_subtypes (
    id UUID PRIMARY KEY,
    product_type_id UUID NOT NULL REFERENCES product_types(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_subtypes_type_id ON product_subtypes (product_type_id);
