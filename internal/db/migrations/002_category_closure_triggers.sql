\c zentora;

-- =============================================================================
-- Category Closure Table – Trigger-based Maintenance
-- =============================================================================
-- Maintains the category_closure (ancestor_id, descendant_id, depth) table
-- automatically whenever rows are inserted, updated (re-parented), or deleted
-- from product_categories.
-- =============================================================================

BEGIN;

-- -----------------------------------------------------------------------
-- Function: fn_maintain_category_closure
-- Called for each INSERT / UPDATE / DELETE on product_categories.
-- -----------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_maintain_category_closure()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- ===========================
    -- INSERT – new category added
    -- ===========================
    IF TG_OP = 'INSERT' THEN
        -- Self-reference (depth = 0)
        INSERT INTO category_closure (ancestor_id, descendant_id, depth)
        VALUES (NEW.id, NEW.id, 0);

        -- Inherit all ancestors from the parent
        IF NEW.parent_id IS NOT NULL THEN
            INSERT INTO category_closure (ancestor_id, descendant_id, depth)
            SELECT ancestor_id, NEW.id, depth + 1
            FROM   category_closure
            WHERE  descendant_id = NEW.parent_id;
        END IF;

        RETURN NEW;
    END IF;

    -- =========================================================================
    -- UPDATE – category re-parented (only when parent_id actually changes)
    -- =========================================================================
    IF TG_OP = 'UPDATE' THEN
        IF OLD.parent_id IS NOT DISTINCT FROM NEW.parent_id THEN
            RETURN NEW; -- parent unchanged, nothing to do
        END IF;

        -- Remove all closure rows that connect an ancestor of the OLD parent
        -- (or OLD.id itself as ancestor) to NEW.id or any of its descendants.
        DELETE FROM category_closure
        WHERE  descendant_id IN (
                   SELECT descendant_id FROM category_closure WHERE ancestor_id = NEW.id
               )
          AND  ancestor_id NOT IN (
                   SELECT descendant_id FROM category_closure WHERE ancestor_id = NEW.id
               );

        -- Re-insert using the new parent's ancestor chain
        IF NEW.parent_id IS NOT NULL THEN
            INSERT INTO category_closure (ancestor_id, descendant_id, depth)
            SELECT p.ancestor_id,
                   c.descendant_id,
                   p.depth + c.depth + 1
            FROM   category_closure p
            CROSS  JOIN category_closure c
            WHERE  p.descendant_id = NEW.parent_id
              AND  c.ancestor_id   = NEW.id;
        END IF;

        RETURN NEW;
    END IF;

    -- =========================================================================
    -- DELETE – category removed (ON DELETE CASCADE on the FK handles the
    -- closure rows automatically via the FK constraint, but we add an explicit
    -- cleanup here for safety when cascades are not in place)
    -- =========================================================================
    IF TG_OP = 'DELETE' THEN
        DELETE FROM category_closure
        WHERE  descendant_id = OLD.id
           OR  ancestor_id   = OLD.id;

        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$;

-- -----------------------------------------------------------------------
-- Trigger: trg_category_closure
-- -----------------------------------------------------------------------
DROP TRIGGER IF EXISTS trg_category_closure ON product_categories;

CREATE TRIGGER trg_category_closure
AFTER INSERT OR UPDATE OF parent_id OR DELETE
ON product_categories
FOR EACH ROW
EXECUTE FUNCTION fn_maintain_category_closure();

COMMIT;
