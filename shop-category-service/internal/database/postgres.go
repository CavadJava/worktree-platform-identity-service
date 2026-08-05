package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"shop-category-service/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

// Migrate owns the `categories` and `subcategories` tables — the
// system-wide catalog every client browses. IDs are human-readable slugs
// (e.g. "women-fashion") rather than UUIDs so seeds stay idempotent and
// clients can ship stable references.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			icon VARCHAR(20),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS subcategories (
			id VARCHAR(100) PRIMARY KEY,
			category_id VARCHAR(100) NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			image VARCHAR(500),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_subcategories_category_id ON subcategories (category_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return seedCategories(db)
}

// seedCategories loads the default catalog (the same 10 categories + 57
// subcategories the taobao-v1 mobile app shipped as mock data) so a fresh
// install is browsable immediately. Idempotent ON CONFLICT DO NOTHING —
// administrator edits are never overwritten.
func seedCategories(db *sql.DB) error {
	type cat struct{ id, name, icon string }
	type sub struct{ id, categoryID, name string }

	cats := []cat{
		{"women-fashion", "Women's Fashion", "👗"},
		{"men-fashion", "Men's Fashion", "👔"},
		{"phones", "Phones & Electronics", "📱"},
		{"home", "Home & Living", "🏠"},
		{"beauty", "Beauty & Personal Care", "💄"},
		{"shoes", "Shoes & Bags", "👜"},
		{"toys", "Toys & Kids", "🧸"},
		{"sports", "Sports & Outdoors", "🏀"},
		{"groceries", "Groceries", "🛒"},
		{"appliances", "Appliances", "🔌"},
	}
	subs := []sub{
		{"wf-plus", "women-fashion", "Women's Plus Size"},
		{"wf-tshirts", "women-fashion", "Women's T-Shirts"},
		{"wf-tanks", "women-fashion", "Tank Tops & Camis"},
		{"wf-shirts", "women-fashion", "Women's Shirts"},
		{"wf-dresses", "women-fashion", "Women's Dresses"},
		{"wf-pants", "women-fashion", "Women's Pants"},
		{"wf-skirts", "women-fashion", "Women's Skirts"},
		{"wf-uv", "women-fashion", "Women's UV Jackets"},
		{"wf-lace", "women-fashion", "Lace & Chiffon Shirts"},
		{"mf-tshirts", "men-fashion", "Men's T-Shirts"},
		{"mf-shirts", "men-fashion", "Men's Shirts"},
		{"mf-sweatpants", "men-fashion", "Men's Sweat Pants"},
		{"mf-jackets", "men-fashion", "Men's Jackets"},
		{"mf-jeans", "men-fashion", "Men's Jeans"},
		{"mf-suits", "men-fashion", "Suits & Blazers"},
		{"ph-cases", "phones", "Phone Cases"},
		{"ph-chargers", "phones", "Chargers & Cables"},
		{"ph-earbuds", "phones", "Earbuds & Audio"},
		{"ph-smartwatch", "phones", "Smart Watches"},
		{"ph-boards", "phones", "DIY Electronics"},
		{"ph-screens", "phones", "Displays & Screens"},
		{"hm-kitchen", "home", "Kitchen & Cooking"},
		{"hm-storage", "home", "Storage & Organizers"},
		{"hm-bedding", "home", "Bedding"},
		{"hm-decor", "home", "Home Decor"},
		{"hm-lighting", "home", "Lighting"},
		{"hm-rugs", "home", "Rugs & Mats"},
		{"bt-skincare", "beauty", "Skincare"},
		{"bt-makeup", "beauty", "Makeup"},
		{"bt-hair", "beauty", "Hair Care"},
		{"bt-fragrance", "beauty", "Fragrance"},
		{"bt-tools", "beauty", "Beauty Tools"},
		{"sh-sneakers", "shoes", "Sneakers"},
		{"sh-sandals", "shoes", "Sandals"},
		{"sh-boots", "shoes", "Boots"},
		{"sh-wallets", "shoes", "Wallets & Card Cases"},
		{"sh-backpacks", "shoes", "Backpacks"},
		{"sh-handbags", "shoes", "Handbags"},
		{"ty-diy", "toys", "DIY Supplies"},
		{"ty-blocks", "toys", "Building Blocks"},
		{"ty-plush", "toys", "Plush Toys"},
		{"ty-baby", "toys", "Baby Supplies"},
		{"ty-puzzles", "toys", "Puzzles & Games"},
		{"sp-fitness", "sports", "Fitness Equipment"},
		{"sp-cycling", "sports", "Cycling"},
		{"sp-camping", "sports", "Camping & Hiking"},
		{"sp-swim", "sports", "Swimming"},
		{"sp-balls", "sports", "Ball Sports"},
		{"gr-snacks", "groceries", "Snacks"},
		{"gr-drinks", "groceries", "Drinks"},
		{"gr-pantry", "groceries", "Pantry Staples"},
		{"gr-fresh", "groceries", "Fresh Food"},
		{"ap-small", "appliances", "Small Appliances"},
		{"ap-vacuum", "appliances", "Vacuums & Cleaning"},
		{"ap-personal", "appliances", "Personal Care Devices"},
		{"ap-aircon", "appliances", "Fans & Air Care"},
	}

	for i, c := range cats {
		if _, err := db.Exec(
			`INSERT INTO categories (id, name, icon, sort_order) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`,
			c.id, c.name, c.icon, i,
		); err != nil {
			return fmt.Errorf("seed category %s: %w", c.id, err)
		}
	}
	for i, s := range subs {
		if _, err := db.Exec(
			`INSERT INTO subcategories (id, category_id, name, image, sort_order) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`,
			s.id, s.categoryID, s.name, "https://picsum.photos/seed/"+s.id+"/300/300", i,
		); err != nil {
			return fmt.Errorf("seed subcategory %s: %w", s.id, err)
		}
	}
	return nil
}
