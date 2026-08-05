DROP TABLE IF EXISTS product_subtypes;
DROP TABLE IF EXISTS product_types;
ALTER TABLE products DROP COLUMN IF EXISTS product_type_id;
