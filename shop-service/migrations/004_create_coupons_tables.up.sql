CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    discount_percent INT NOT NULL,
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_coupons (
    user_id UUID NOT NULL,
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, coupon_id)
);

CREATE INDEX IF NOT EXISTS idx_user_coupons_user_id ON user_coupons (user_id);
