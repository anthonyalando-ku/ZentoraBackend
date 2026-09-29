-- Apply once, using the same database/search_path as the API, before backend deployment.
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE product_categories ADD COLUMN image_url TEXT NULL;
COMMIT;
