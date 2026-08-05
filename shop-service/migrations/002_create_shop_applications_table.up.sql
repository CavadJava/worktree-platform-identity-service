CREATE TABLE IF NOT EXISTS shop_applications (
    id UUID PRIMARY KEY,
    applicant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_version VARCHAR(20) NOT NULL DEFAULT 'v1',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewer_id UUID,
    review_note TEXT,
    shop_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shop_applications_status ON shop_applications (status);
CREATE INDEX IF NOT EXISTS idx_shop_applications_applicant ON shop_applications (applicant_id);
