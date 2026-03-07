# Product Discovery Engine Architecture

This document finalizes the architecture for Zentora's product discovery engine before SQL implementation begins. It is grounded in the current PostgreSQL schema under `/home/runner/work/ZentoraBackend/ZentoraBackend/internal/db/migrations/001_svc_init.sql` and proposes only the additional structures needed to support production-grade feeds, search, autosuggest, and personalization.

## 1. Goals and Architectural Principles

The discovery engine must power:

- homepage sections
- category browsing
- product search
- search autosuggestions
- personalized recommendations
- trending and deal feeds
- editorial and merchandising sections

The design should be:

- **modular**: feed generation, filtering, ranking, and response assembly are separate concerns
- **reusable**: all discovery surfaces reuse the same retrieval pipeline
- **performant**: heavy signals are precomputed; online queries stay small and index-friendly
- **extensible**: new ranking signals can be added without rewriting every feed

## 2. Current Schema Fit

The existing schema already provides most of the core discovery signals:

### Product and merchandising data
- `products`
- `product_variants`
- `product_images`
- `product_category_map`
- `product_brands`
- `tags`
- `product_tags`
- `homepage_sections`
- `banners`

### Category hierarchy
- `product_categories`
- `category_closure`

### Inventory and price inputs
- `inventory_items`
- `discounts`
- `discount_targets`
- `discount_redemptions`

### Behavioral and quality signals
- `product_events`
- `reviews`
- `wishlists`
- `wishlist_items`
- `order_items`
- `user_category_affinity`
- `product_co_views`
- `product_metrics`

This means the first implementation step should not be a redesign of the catalog schema. Instead, discovery should be added as a layer on top of the current schema with a few targeted schema additions for search logging and search indexing.

## 3. High-Level System Architecture

The discovery engine should be implemented as four cooperating layers.

### A. Online request layer
A new discovery module should expose a single service entry point such as:

- `GetFeed(ctx, FeedRequest)`
- `Search(ctx, SearchRequest)`
- `Suggest(ctx, SuggestRequest)`
- `TrackSearch(ctx, SearchEvent)`
- `TrackSearchClick(ctx, SearchClickEvent)`

Recommended package layout:

- `/home/runner/work/ZentoraBackend/ZentoraBackend/internal/service/discovery`
- `/home/runner/work/ZentoraBackend/ZentoraBackend/internal/repository/postgres/discovery_repo.go`
- `/home/runner/work/ZentoraBackend/ZentoraBackend/internal/handlers/discovery`

### B. Retrieval layer
This layer is responsible for producing candidate product IDs from multiple strategies:

- trending products from `product_metrics`
- best sellers from `order_items` and `product_metrics.weekly_purchases`
- recommended products from `product_events`, `wishlist_items`, `order_items`, `user_category_affinity`, and `product_co_views`
- deal candidates from `discounts` and `discount_targets`
- featured/editorial candidates from `products.is_featured` and `homepage_sections`
- category candidates from `product_category_map` expanded through `category_closure`
- high-rating candidates from `products.rating` and `reviews`
- freshness candidates from `products.created_at`

Each retrieval strategy should return a bounded set of candidate IDs plus raw source scores. The retrieval layer should not apply frontend formatting.

### C. Filtering and eligibility layer
This layer applies hard constraints that remove products before ranking:

- `products.status = 'active'`
- category membership including descendants via `category_closure`
- optional brand filters
- optional tag filters
- price range filters
- minimum rating filters
- discount-only filters
- in-stock filters based on aggregate available inventory
- variant attribute filters based on `variant_attribute_values` and `attribute_values`

This layer should also normalize inventory into three frontend states:

- `in_stock`
- `low_stock`
- `out_of_stock`

### D. Ranking and response layer
The ranking layer combines reusable signals into a final score. The response layer then hydrates only the winning product IDs into frontend-ready cards containing:

- `product_id`
- `name`
- `slug`
- `primary_image`
- `price`
- `discount`
- `rating`
- `review_count`
- `inventory_status`
- `brand`
- `category`

Hydration should happen after ranking so joins to images, brands, and categories are only done for the top N results.

## 4. Retrieval Pipeline

All feeds should use the same 3-stage pipeline.

### Stage 1: Candidate retrieval
Candidate retrieval should be feed-specific but structurally identical.

| Feed type | Primary candidate sources |
| --- | --- |
| `trending` | `product_metrics.trending_score`, recent `product_events` |
| `best_sellers` | `product_metrics.weekly_purchases`, `order_items` |
| `recommended` | `user_category_affinity`, `product_co_views`, `wishlist_items`, `order_items`, `product_events` |
| `category` | `product_category_map` + descendant categories from `category_closure` |
| `deals` | active `discounts` + `discount_targets` |
| `new_arrivals` | `products.created_at` |
| `highly_rated` | `products.rating`, `products.review_count`, `reviews` |
| `most_wishlisted` | `wishlist_items` counts |
| `also_viewed` | `product_co_views` |
| `featured` | `products.is_featured`, `homepage_sections` |
| `editorial` | `homepage_sections` plus curated product IDs |
| `search` | text index candidates + prefix/typo matches |

Recommended retrieval policy:

- fetch from multiple sources per feed
- union candidates into a de-duplicated pool
- cap the pool to a manageable size such as 200-1000 IDs depending on endpoint
- preserve source-level feature values for ranking

### Stage 2: Filtering
Filtering should be deterministic and shared by all feeds. The filter order should prefer cheap eliminations first:

1. active product status
2. category closure expansion
3. inventory availability
4. price and brand filters
5. tag filters
6. rating and discount filters
7. variant attribute filters

This ordering prevents expensive joins for already-ineligible products.

### Stage 3: Ranking
Use a weighted linear scorer first, with per-feed weight overrides. A strong default is:

`FinalScore = 0.35*personalization + 0.25*trending + 0.15*conversion + 0.10*rating + 0.10*discount + 0.05*freshness`

The ranker should accept a signal map so new features can be added without rewriting candidate retrieval. Example signal names:

- `text_relevance`
- `personalization_score`
- `trending_score`
- `popularity_score`
- `conversion_rate`
- `rating_score`
- `discount_score`
- `freshness_score`
- `inventory_score`
- `merchandising_score`

## 5. Ranking Signals

Signals should be normalized to a shared range, ideally `[0,1]`, before weighting.

### Core reusable signals

| Signal | Source | Notes |
| --- | --- | --- |
| `text_relevance` | search index score | Only used for search/autosuggest ranking |
| `trending_score` | `product_metrics.trending_score` | Primary short-term demand signal |
| `popularity_score` | `daily_views`, `weekly_views`, wishlist counts, purchases | Good fallback for anonymous users |
| `conversion_rate` | `product_metrics.conversion_rate` | Helps rank products that sell after being viewed |
| `rating_score` | `products.rating` + `products.review_count` | Use Bayesian smoothing to avoid small-sample bias |
| `discount_score` | active discount percentage or effective price delta | Use 0 if no active discount |
| `freshness_score` | `products.created_at` | Decay over time |
| `inventory_score` | sum of `available_qty - reserved_qty` | Prevent ranking unavailable items too high |
| `personalization_score` | user/category/product affinity | Higher for returning users |
| `co_view_score` | `product_co_views.score` | Used heavily in also-viewed and recommendation feeds |
| `merchandising_score` | `is_featured`, `homepage_sections`, banner priority | Lets business curate without hard-coding lists |

### Feed-specific weighting guidance
- **Homepage trending**: favor `trending_score`, `conversion_rate`, `inventory_score`
- **Deals**: favor `discount_score`, `conversion_rate`, `rating_score`
- **Category pages**: favor `popularity_score`, `rating_score`, `inventory_score`
- **Personalized feed**: favor `personalization_score`, `co_view_score`, `conversion_rate`
- **Search**: favor `text_relevance`, then business/relevance tie-breakers

## 6. Personalization Strategy

The personalization policy should be audience-specific.

### Anonymous users
Use non-user-specific signals only:

- trending products
- best sellers
- high-conversion products
- high-inventory products
- active deals

If a `session_id` exists, session-level recently viewed products can still power `also_viewed` and `recently_seen` recovery.

### New registered users
When a user has little to no behavioral history:

- use global trending and best sellers as the base
- blend in category popularity from first-session browsing
- optionally use first-party onboarding choices later, if added

### Returning users
For users with interaction history, build personalization from:

- recent product views from `product_events`
- purchases from `order_items`
- wishlist additions from `wishlist_items`
- category preference from `user_category_affinity`
- similar-item expansion from `product_co_views`

Recommended scoring recipe for personalized candidates:

- compute category preference scores from views, cart events, purchases, and wishlist additions
- boost products in top affinity categories
- boost products co-viewed with recently viewed or purchased items
- down-rank already purchased items when the feed is meant for discovery rather than replenishment
- keep a minimum share of globally trending products to avoid overfitting and to help exploration

## 7. Search Architecture

Search should be implemented as a two-phase retrieval and re-ranking flow.

### Search inputs to match
- product name
- product description
- tag names
- brand name
- category names

### Recommended PostgreSQL strategy
Use PostgreSQL native search with two complementary indexes:

1. **Full-text search** for relevance ranking
2. **`pg_trgm` trigram indexes** for partial matches, prefix matches, and typo tolerance

### Search document
Create a denormalized searchable document per product containing:

- weighted product name
- weighted short description and description
- brand name
- category names
- tag names

Recommended weighting:

- name: highest weight
- brand and categories: medium weight
- tags: medium weight
- descriptions: lower weight

This can live in either:

- a materialized view such as `product_search_documents`, or
- a dedicated table refreshed by triggers/jobs

### Search flow
1. normalize the query
2. retrieve candidates from FTS and trigram matching
3. union and de-duplicate candidates
4. apply catalog filters
5. compute `SearchScore`
6. return hydrated product cards
7. log the search event and result positions

### Initial search ranking formula
`SearchScore = 0.50*text_relevance + 0.20*popularity + 0.15*conversion_rate + 0.10*rating + 0.05*trending`

This formula should be configurable so weights can be tuned from real click-through data later.

## 8. Autosuggest Design

Autosuggest must return in under 100 ms, so it should not depend on large multi-join online queries.

### Suggestion types
- categories
- products
- brands
- popular past queries

### Recommended design
Use a compact suggestion index populated from:

- `products.name`
- `product_categories.name`
- `product_brands.name`
- historical search queries from `search_events`

For performance, maintain a dedicated suggestions table or Redis cache keyed by normalized prefix. Each suggestion record should include:

- suggestion text
- suggestion type
- normalized prefix tokens
- popularity score
- optional reference ID

### Autosuggest ranking
Order suggestions using:

- exact prefix match quality
- historical query frequency
- recent trend score
- click-through rate for suggestion -> product/search result
- business priority for featured categories or brands

### Latency strategy
- precompute prefix candidates offline or on write
- cache hot prefixes in Redis
- cap suggestions per type
- avoid joining the full product tables during suggestion lookup

## 9. Search Event Tracking

To support future learning-to-rank and suggestion quality improvements, search interactions need first-class logging.

### Required online writes
For every search request, write to `search_events`:

- `query`
- `normalized_query`
- `user_id`
- `session_id`
- `result_count`
- `created_at`

For every clicked result, write to `search_clicks`:

- `search_event_id`
- `product_id`
- `position`
- `user_id`
- `session_id`
- `created_at`

To support offline relevance analysis, persist the ranked set in `search_result_positions`:

- `search_event_id`
- `product_id`
- `position`
- `score`

## 10. Indexing Strategy

### Existing indexes that discovery can already use
- `idx_products_status`
- `idx_products_featured`
- `idx_products_brand`
- `idx_products_created_at`
- `idx_pcm_category`
- `idx_product_images_product`
- `idx_variants_product`
- `idx_inventory_variant`
- `idx_product_events_product_time`
- `idx_product_events_user_time`
- `idx_product_events_type_time`
- `idx_product_metrics_trending`
- `idx_reviews_product`
- `idx_category_closure_desc`

### Additional indexes required

#### Catalog and feed retrieval
- `CREATE INDEX idx_category_closure_ancestor_depth ON category_closure(ancestor_id, depth, descendant_id);`
- `CREATE INDEX idx_pcm_category_product ON product_category_map(category_id, product_id);`
- `CREATE INDEX idx_product_tags_tag_product ON product_tags(tag_id, product_id);`
- `CREATE INDEX idx_inventory_variant_available ON inventory_items(variant_id, available_qty, reserved_qty);`
- `CREATE INDEX idx_wishlist_items_product_added ON wishlist_items(product_id, added_at);`
- `CREATE INDEX idx_order_items_product_created ON order_items(product_id, created_at);`
- `CREATE INDEX idx_user_category_affinity_user_score ON user_category_affinity(user_id, score DESC);`
- `CREATE INDEX idx_product_co_views_product_score ON product_co_views(product_id, score DESC);`
- `CREATE INDEX idx_product_metrics_purchases ON product_metrics(weekly_purchases DESC);`
- `CREATE INDEX idx_product_metrics_conversion ON product_metrics(conversion_rate DESC);`

#### Search
- trigram GIN indexes on product names, brand names, category names, and query history
- GIN index on the search document `tsvector`
- `CREATE INDEX idx_search_events_created_query ON search_events(created_at, normalized_query);`
- `CREATE INDEX idx_search_clicks_event_position ON search_clicks(search_event_id, position);`
- `CREATE INDEX idx_search_result_positions_event_position ON search_result_positions(search_event_id, position);`

#### Partitioning recommendation
- partition `product_events` by time when volume grows
- partition `search_events` and `search_clicks` by time from day one if search traffic is expected to be large

## 11. Schema Improvements Recommended Before Implementation

The following schema additions are the minimum recommended improvements.

### A. Search logging tables
Add:

- `search_events`
- `search_clicks`
- `search_result_positions`

These tables are required to satisfy the product requirements and to support relevance tuning.

### B. Search document storage
Add either:

- `product_search_documents(product_id, document, search_vector, updated_at)`, or
- a materialized view with equivalent fields

This avoids rebuilding joins across products, brands, tags, and categories on every search.

### C. Stronger relational guarantees
Add foreign keys for:

- `product_co_views.product_id -> products.id`
- `product_co_views.related_product_id -> products.id`
- `user_category_affinity.user_id -> auth_identities.id`
- `user_category_affinity.category_id -> product_categories.id`

These tables are already useful, but relational guarantees will improve data quality.

### D. Discovery-friendly denormalizations
Consider adding or materializing:

- `effective_price`
- `active_discount_percent`
- `primary_category_id`
- `primary_image_url`
- aggregated `available_inventory`

These values can remain derived at first, but materializing them will simplify ranking and reduce join cost on hot endpoints.

### E. Enumerated event types
Replace free-text event strings with enums or constrained checks for:

- `product_events.event_type`
- `discount_targets.target_type`
- search suggestion types if a suggestion table is added

This reduces bad data and makes analytics more reliable.

## 12. Feed Execution Model

Every feed request should follow the same contract.

### Inputs
- `feed_type`
- `user_id`
- `session_id`
- `category_id`
- `query`
- `limit`
- `filters`

### Output contract
Return a stable product card DTO with:

- `product_id`
- `name`
- `slug`
- `primary_image`
- `price`
- `discount`
- `rating`
- `review_count`
- `inventory_status`
- `brand`
- `category`

### Shared flow
1. choose retrieval strategies from `feed_type`
2. collect and union candidate IDs
3. apply shared filters
4. rank with feed-specific weights
5. hydrate top products
6. emit analytics events where relevant

This contract keeps homepage sections, category pages, search, and recommendation widgets on the same engine.

## 13. Recommended Implementation Sequence

After this architecture is accepted, implementation should proceed in this order:

1. add the search logging and search-document schema changes
2. add a `discovery` repository/service with shared request and filter models
3. implement non-personalized feeds first: trending, best sellers, deals, new arrivals, featured
4. implement category retrieval using `category_closure`
5. implement PostgreSQL search and autosuggest
6. add search tracking writes
7. implement personalized ranking using `user_category_affinity` and `product_co_views`
8. add Redis caching for hot feeds and hot prefixes

This sequence minimizes risk while producing usable discovery endpoints early.
