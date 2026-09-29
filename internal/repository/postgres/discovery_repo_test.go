package postgres

import (
	"context"
	"strings"
	"testing"

	discoverydomain "zentora-service/internal/domain/discovery"
)

func TestHydrateProductCardsReturnsEmptyForNoIDs(t *testing.T) {
	repo := &DiscoveryRepository{}

	got, err := repo.HydrateProductCards(context.Background(), nil)
	if err != nil {
		t.Fatalf("HydrateProductCards() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("HydrateProductCards() = %#v, want empty result", got)
	}
}

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
	}, nil)
	if len(args) != 9 {
		t.Fatalf("args len = %d, want 9", len(args))
	}

	ids, ok := args[7].([]int64)
	if !ok {
		t.Fatalf("variant attribute args type = %T, want []int64", args[7])
	}
	if len(ids) != 2 || ids[0] != 13 || ids[1] != 21 {
		t.Fatalf("variant attribute args = %#v, want [13 21]", ids)
	}
}

func TestEligibleCategoryIDNarrowsRankedFeedsOnly(t *testing.T) {
	categoryID := int64(7)

	trending := &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedTrending, CategoryID: &categoryID}
	if got := eligibleCategoryID(trending); got == nil || *got != categoryID {
		t.Fatalf("trending category = %v, want %d", got, categoryID)
	}
	args := buildEligibleProductsArgs(trending.Filters, eligibleCategoryID(trending))
	if got, ok := args[8].(*int64); !ok || got == nil || *got != categoryID {
		t.Fatalf("category arg = %#v, want %d", args[8], categoryID)
	}

	// The category feed already includes descendant categories; an exact
	// eligibility match would wrongly drop products from child categories.
	category := &discoverydomain.FeedRequest{FeedType: discoverydomain.FeedCategory, CategoryID: &categoryID}
	if got := eligibleCategoryID(category); got != nil {
		t.Fatalf("category feed eligibility category = %d, want nil", *got)
	}
}

func TestBuildEditorialCandidateQueryUsesHomepageSections(t *testing.T) {
	query := buildEditorialCandidateQuery()

	expectedFragments := []string{
		"homepage_sections",
		"custom_sections",
		"featured_sections",
		"category_sections",
		"trending_sections",
		"type = 'custom'",
		"type = 'featured'",
		"type = 'category'",
		"type = 'trending'",
		"product_metrics",
		"category_closure",
		"merchandising_score",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected query to contain %q, got:\n%s", fragment, query)
		}
	}
}

func TestBuildFeaturedCandidateQueryUsesHomepageSections(t *testing.T) {
	query := buildFeaturedCandidateQuery()

	expectedFragments := []string{
		"homepage_sections",
		"featured_section_products",
		"type IN ('featured', 'custom')",
		"p.is_featured = TRUE",
		"merchandising_score",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected query to contain %q, got:\n%s", fragment, query)
		}
	}
}
