#!/usr/bin/env bash
# Idempotent seed script for the "Taobao Marketplace" shop that taobao-v1's
# checkout flow depends on (see taobao-v1/src/api/checkout.ts and
# taobao-v1/src/api/marketplace.ts). The mobile app's browsing catalog is
# bundled mock data (taobao-v1/src/data/products.ts, ids p1..p10); checkout
# resolves each mock id to a real backend product tagged `mock:<id>` in its
# description, under a shop literally named "Taobao Marketplace".
#
# This mapping used to be created by hand, once, with no script and no
# migration — any fresh backend (or a partially-seeded one, e.g. missing a
# single product) breaks checkout with "None of the cart items exist on the
# backend". This script is safe to re-run any time: it only creates what's
# missing (shop, product, or variants), so re-running after a partial seed
# just fills the gap instead of duplicating anything.
#
# Usage:
#   ./scripts/seed-taobao-marketplace.sh
#   ADMIN_EMAIL=other@example.com ADMIN_PASSWORD=secret ./scripts/seed-taobao-marketplace.sh
#
# Requires: curl, jq. Requires registration-service, shop-service and
# shop-product-service running locally (default ports below).

set -euo pipefail

REGISTRATION_URL="${REGISTRATION_URL:-http://localhost:8081/api/v1}"
SHOP_URL="${SHOP_URL:-http://localhost:8086/api/v1}"
SHOP_PRODUCT_URL="${SHOP_PRODUCT_URL:-http://localhost:8087/api/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-password123}"
MARKETPLACE_SHOP_NAME="Taobao Marketplace"

# Mirrors taobao-v1/src/data/products.ts exactly (id, name, price). If that
# file's catalog changes, update this list to match.
MOCK_PRODUCTS_TSV="p1	Oversized Knit Sweater — Autumn/Winter	12.90
p2	Men's Slim Fit Casual Shirt	9.50
p3	Wireless Earbuds Bluetooth 5.3	14.99
p4	Ceramic Non-Stick Frying Pan 28cm	11.20
p5	Vitamin C Brightening Serum 30ml	6.80
p6	Canvas Tote Bag — Minimalist	5.50
p7	Building Blocks Set — 500 pcs	8.90
p8	Adjustable Dumbbell Set 2x5kg	19.90
p9	Hyaluronic Acid Moisturizing Cream 50g	8.20
p10	Clear Shockproof Phone Case	3.90"

# Matches the variant set every already-seeded product carries — see
# taobao-v1/src/api/checkout.ts's BuyNowSheet variant list.
VARIANT_NAMES="Beige Black Gray Navy White"

log() { echo "[seed] $*"; }

log "Logging in as ${ADMIN_EMAIL}..."
LOGIN_RESPONSE=$(curl -sf -X POST "${REGISTRATION_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}")
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "[seed] ERROR: login failed: $LOGIN_RESPONSE" >&2
  exit 1
fi
AUTH=(-H "Authorization: Bearer ${TOKEN}")

log "Looking up shop \"${MARKETPLACE_SHOP_NAME}\"..."
SHOP_ID=$(curl -sf "${SHOP_URL}/shops" | jq -r --arg name "$MARKETPLACE_SHOP_NAME" \
  '.data[] | select(.name == $name) | .id' | head -n1)

if [ -z "$SHOP_ID" ]; then
  log "Not found — creating it..."
  SHOP_ID=$(curl -sf -X POST "${SHOP_URL}/shops" "${AUTH[@]}" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${MARKETPLACE_SHOP_NAME}\",\"description\":\"Mirrors taobao-v1's bundled mock catalog for real checkout/chat.\"}" \
    | jq -r '.data.id')
  log "Created shop ${SHOP_ID}"
else
  log "Found shop ${SHOP_ID}"
fi

log "Fetching existing products for this shop..."
EXISTING_PRODUCTS=$(curl -sf "${SHOP_PRODUCT_URL}/products?shop_id=${SHOP_ID}")

created_products=0
created_variants=0
skipped=0

while IFS=$'\t' read -r mock_id name price; do
  [ -z "$mock_id" ] && continue

  PRODUCT_ID=$(echo "$EXISTING_PRODUCTS" | jq -r --arg d "mock:${mock_id}" \
    '.data[] | select(.description == $d) | .id' | head -n1)

  if [ -z "$PRODUCT_ID" ]; then
    log "Product ${mock_id} (\"${name}\") missing — creating..."
    PRODUCT_ID=$(curl -sf -X POST "${SHOP_PRODUCT_URL}/products" "${AUTH[@]}" \
      -H "Content-Type: application/json" \
      -d "{\"shop_id\":\"${SHOP_ID}\",\"name\":$(jq -Rn --arg s "$name" '$s'),\"description\":\"mock:${mock_id}\",\"price\":${price},\"stock\":999}" \
      | jq -r '.data.id')
    created_products=$((created_products + 1))
  else
    log "Product ${mock_id} (\"${name}\") already exists (${PRODUCT_ID})"
  fi

  VARIANT_COUNT=$(curl -sf "${SHOP_PRODUCT_URL}/products/${PRODUCT_ID}/items" | jq -r '.data | length')
  if [ "$VARIANT_COUNT" = "0" ]; then
    log "  No variants — creating ${VARIANT_NAMES}..."
    for variant in $VARIANT_NAMES; do
      curl -sf -X POST "${SHOP_PRODUCT_URL}/products/${PRODUCT_ID}/items" "${AUTH[@]}" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"${variant}\",\"price\":${price},\"stock\":999,\"is_discounted\":false}" \
        > /dev/null
      created_variants=$((created_variants + 1))
    done
  else
    skipped=$((skipped + 1))
  fi
done <<< "$MOCK_PRODUCTS_TSV"

log "Done. Shop: ${SHOP_ID} | products created: ${created_products} | variant sets already present: ${skipped} | variants created: ${created_variants}"
