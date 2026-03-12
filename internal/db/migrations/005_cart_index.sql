\c zentora;

BEGIN;
CREATE UNIQUE INDEX IF NOT EXISTS ux_carts_user_active
ON carts(user_id)
WHERE status = 'active';

-- Variant is required for cart_items (matches your "inventory tracking via variants" rule).
ALTER TABLE cart_items
  ALTER COLUMN variant_id SET NOT NULL;

COMMIT;