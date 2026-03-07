package postgres

import (
	"strings"
	"testing"

	discoverydomain "zentora-service/internal/domain/discovery"
)

func TestBuildEligibleProductsCTEIncludesSharedFilterClauses(t *testing.T) {
	sql := buildEligibleProductsCTE(3)

	expectedFragments := []string{
		"inventory_summary",
		"best_discounts",
		"product_tags",
		"variant_attribute_values",
		"available_inventory",
		"discount_percent",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got:\n%s", fragment, sql)
		}
	}
}

func TestBuildEligibleProductsArgsIncludesVariantAttributeIDs(t *testing.T) {
	args := buildEligibleProductsArgs(discoverydomain.FeedFilter{
		VariantAttributeValueIDs: []int64{13, 21},
	})
	if len(args) != 8 {
		t.Fatalf("args len = %d, want 8", len(args))
	}

	ids, ok := args[7].([]int64)
	if !ok {
		t.Fatalf("variant attribute args type = %T, want []int64", args[7])
	}
	if len(ids) != 2 || ids[0] != 13 || ids[1] != 21 {
		t.Fatalf("variant attribute args = %#v, want [13 21]", ids)
	}
}
