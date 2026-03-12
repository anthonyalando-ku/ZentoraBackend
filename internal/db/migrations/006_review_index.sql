\c zentora;
BEGIN ;
-- Align DB constraint to "one review per order item" (recommended).
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_user_id_product_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS ux_reviews_user_order_item ON reviews(user_id, order_item_id);

-- Optional: still prevent duplicate reviews for same product without order reference
-- (depends on your business rules).
-- CREATE UNIQUE INDEX IF NOT EXISTS ux_reviews_user_product ON reviews(user_id, product_id);
COMMIT ;