BEGIN;

CREATE INDEX IF NOT EXISTS idx_product_tags_tag_product
    ON product_tags(tag_id, product_id);

CREATE INDEX IF NOT EXISTS idx_variant_attribute_values_attr_value_variant
    ON variant_attribute_values(attribute_value_id, variant_id);

CREATE INDEX IF NOT EXISTS idx_inventory_variant_available
    ON inventory_items(variant_id, available_qty, reserved_qty);

COMMIT;
