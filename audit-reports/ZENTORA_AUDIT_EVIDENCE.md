# Zentora audit evidence register
Audit date: 22 September 2026. Backend root **B**: D:/Projects/ZentoraBackend. Frontend root **F**: D:/Projects/ZentoraUI. References below use root-relative paths and one-based line numbers. Source observations concern the local working trees, not a verified deployed revision.

## Method and limits
Read-only repository discovery, targeted cross-layer traces, configuration review, offline Go tests, and public web extraction. No production database connection, login, account creation, order, payment, admin request, migration or deployment was performed. Only audit documentation was created. No browser automation tool was available: responsive layout, interactive checkout, accessibility and performance require browser verification. Dependency versions were inventoried; a current vulnerability scan was not completed. Environment secret values were not inspected or copied. Tracked configs/local.env and configs/prod.env are empty; that is not evidence that all repository history is secret-free.

A feature is **implemented** when connected source supports it, **partial** when workflow steps are incomplete, **configured/unverified** when configuration or schema exists without runtime proof, **missing** when no connected implementation was found across routes, services, repositories and UI, **broken** when a source contradiction demonstrates incorrect behavior, and **N/A** when not selected for current scope. Missing always means missing from the reviewed connected application, not a claim about unknown external operations.

## Trace register
| ID | Evidence | Conclusion |
|---|---|---|
| E01 | B/go.mod:3,8; F/package.json:6,14 | Go 1.24.5 declaration, Gin 1.11; React 19, Vite 7, TypeScript 5, TanStack Query, Axios, Zustand |
| E02 | B/internal/app/server.go:155,187,195,203,244,266; B/internal/app/router.go:39 | Modular monolith, PostgreSQL repositories, Redis, in-process metrics, HTTP routing |
| E03 | B/internal/app/router.go:113; B/internal/handlers/order/handler.go:66,85,103,184; B/internal/service/order/order_get.go:10; B/internal/repository/postgres/order.go:91,239 | Authenticated order detail lacks owner predicate; list accepts client user filter or no filter |
| E04 | B/internal/service/order/service.go:265,302,311,321,325,346; B/internal/repository/postgres/order.go:43 | Server variant prices, transactional order/stock write; float money, zero discount/tax and no delivery quote |
| E05 | B/internal/service/order/service.go:171,176; B/internal/service/order/order_mngt.go:27; B/internal/repository/postgres/order.go:323 | Cart conversion outside transaction; unrestricted valid status changes; cancellation has no inventory release |
| E06 | B/internal/repository/postgres/inventory_repo.go:267,283,312; B/internal/service/catalog/inventory.go:116 | Atomic reservation and manual release exist; release is not tied to order lifecycle |
| E07 | F/src/features/checkout/pages/CheckoutPage.tsx:251,412,437,486; B/internal/service/order/service.go:119; B/internal/db/migrations/001_svc_init.sql:359 | COD UI enabled; M-Pesa disabled; requested payment method discarded; payment table alone is not an integration |
| E08 | B/internal/service/order/service.go:130,181,195,201; B/cmd/worker/main.go:39; B/internal/service/worker/order_reminder_job.go:33 | Customer confirmation path receives blank email or is commented; separate worker supports admin reminders |
| E09 | B/internal/merchant/handler/handler.go:48,95,104; B/internal/app/router.go:45,310 | Public Google feed plus admin routes wired; public request regenerates full feed before ETag comparison |
| E10 | B/internal/merchant/infrastructure/postgres/merchant_repo.go:325,358; F/src/features/products/pages/ProductDetailPage.tsx:68,96 | Feed IDs productID-variantID and variant landing URL; frontend supports variant parameter |
| E11 | B/internal/merchant/domain/merchant_domain.go:251; B/internal/merchant/infrastructure/postgres/queries.go:196,212,588; E06 | Feed subtracts reserved from already-decremented available stock |
| E12 | B/internal/merchant/infrastructure/postgres/queries.go:82,90; B/internal/merchant/application/feedgen/generator.go:91,123,148 | Feed discount resolution exists; incremental gate only sees product update/variant creation, not independent inventory changes |
| E13 | F/index.html:147; source-wide searches of F/src, F/index.html and B/internal | Google Ads base tag present; no GA4 config/ecommerce dispatch, click-ID capture or consent UI found in connected frontend |
| E14 | B/internal/app/router.go:205; B/internal/repository/postgres/discovery_queries.go:17,145; F/src/core/api/services/discoverySearch.ts:66,80,85,90 | Search, suggestions, fuzzy ranking and first-party search/click events exist |
| E15 | B/internal/db/migrations/001_svc_init.sql:75,154,177,203,240,306,443,491; B/internal/db/migrations/000_auth_init.sql:23 | Baseline domain tables, indexes, price snapshots, media, discounts, events and banner scaffolding |
| E16 | B/internal/db/migrations/merged_schema.sql:17,26,1191,1227,1252,1422,1586; B/internal/db/migrations/missing.sql:1142 | Alternative expanded schemas include sellers/payments/consent; no proof of migration or connected application use |
| E17 | B/scripts/migrate.sh (0 bytes); B/build/ci-cd.yml (0 bytes); numbered migrations include duplicate 005 prefix | Migration and CI placeholders; no runnable GitHub Actions workflow found in discovery |
| E18 | B/docker-compose.yaml:18; B/Dockerfile:1,7,12,20; B/render.yaml:1; F/netlify.toml:1 | API+Redis Compose, external PostgreSQL, nonroot Debian image; Render and Netlify deployment proposals coexist |
| E19 | B/render.yaml:15; B/internal/config/config.go:53; B/internal/db/redis.go:44 | Render supplies REDIS_URL while connected client uses REDIS_ADDR/REDIS_PASS; deployment needs reconciliation |
| E20 | B/internal/middleware/auth_middleware.go:25,225,277; B/internal/service/auth/auth.go:108,210,580; B/internal/repository/postgres/auth_repo.go:288 | Bcrypt, login throttling, JWT+session checks, role guards; DB session lookup filters active/unexpired |
| E21 | B/internal/middleware/auth_middleware.go:287; B/internal/middleware/logging_middleware.go:14,26; F/src/core/api/token.ts:1 | Query bearer tokens can enter raw-query logs; localStorage bearer tokens increase impact of XSS |
| E22 | B/internal/middleware/cors_middleware.go:10; B/internal/pkg/response/response.go:32; B/internal/app/server.go:266 | Wildcard credentialed CORS, raw error responses and absent explicit server timeout fields |
| E23 | B/internal/service/catalog/image_helper.go:83,105,119,150; B/internal/handlers/catalog/product.go:20 | Image decoding/re-encoding and local fallback; no explicit byte/pixel cap in image helper; deployment persistence must be verified |
| E24 | F/src/app/router/adminRoutes.tsx:49; B/internal/app/router.go:127,234; F/src/features/admin | Catalog, discounts, inventory, orders and dashboard management exist |
| E25 | F/src/features/public/home/components/HeroMarketPlace.tsx:7; F/src/features/public/home/pages/HomePage.tsx:73; B/internal/service/homapage/service.go:71 | Active hero slides hard-coded; homepage service exists but no wired homepage CMS handler found |
| E26 | F/src/features/cart/hooks/useCart.ts:39,80,109; B/internal/service/cart/service.go:68; B/internal/repository/postgres/cart.go:59,163 | Guest/local and authenticated/server cart; submitted cart price stored, final order price recomputed |
| E27 | B/internal/service/review/service.go:24,37,47; B/internal/app/router.go:158,194 | Purchase-linked review ownership/status checks exist |
| E28 | F/src/shared/seo/ProductJsonLd.tsx:1; F/netlify/edge-functions/product-og.ts; F/public/sitemap-index.xml | Product structured data, crawler edge code and sitemaps exist; full crawl consistency unverified |
| E29 | B/internal/repository/postgres/order.go:388 | Dashboard labels completed/delivered order sums as revenue without payment/refund ledger |
| E30 | B/internal/repository/postgres/auth_repo.go:574 | Permission query combines OR and AND without grouping both grant branches under revoke exclusion |
| E31 | B/internal/db/postgres.go:14,50; B/internal/pkg/session/manager.go:60,216; B/docker-compose.yaml:33 | TLS requested, pool limit 25; Redis session cache plus blacklist and allkeys-lru eviction policy |
| E32 | B/internal/service/discovery/service_test.go:265; B/internal/service/worker/metrics_jobs_test.go:36; B/internal/handlers/discovery/handler_test.go:68; B/internal/repository/postgres/discovery_repo_test.go:11 | Existing tests concentrated on discovery, validation and metrics, not commerce transactions |
| E33 | F/src/app/router/publicRoutes.tsx:30; F/src/features/public/pages/PrivacyPolicyPage.tsx:243,281 | Customer policy/support pages exist; optional-cookie promises require implementation alignment |

## Public observations
Web extraction returned homepage title, navigation, business contact information and delivery/returns promises from [public homepage](https://zentorashop.co.ke/). Extraction of /returns returned the same small shell, not proof that browser rendering fails. /products, /cart and /checkout returned tool retrieval errors. No conclusion about production availability follows from those errors. No product URL was available for a reliable live product inspection. Merchant feed runtime, Ads campaigns/account links, consent behavior and production payment behavior remain unverified. User confirms Google Merchant Center/Shopping and Google Ads are in use; third-party sellers are future scope.

## External references consulted
- [Google ecommerce events](https://developers.google.com/analytics/devguides/collection/ga4/ecommerce): event names, item payload and currency guidance.
- [Google SPA pageviews](https://developers.google.com/analytics/devguides/collection/ga4/views): avoid duplicate automatic/manual measurement.
- [Google consent implementation](https://developers.google.com/tag-platform/security/guides/consent): technical defaults and consent updates.
- [Merchant product specification](https://support.google.com/merchants/answer/7052112?hl=en-GB): product data contract.
- [Merchant structured data](https://support.google.com/merchants/answer/7331077?hl=en-GB): price/availability alignment.
- [Paystack Kenya pricing](https://paystack.com/ke/pricing): published M-Pesa 1.5%, local cards 2.9%, international cards 3.8%, T+2 settlement. Commercial terms must be reconfirmed before contracting.
- [Paystack webhook documentation](https://paystack.com/docs/payments/webhooks/) and [verification](https://paystack.com/docs/payments/verify-payments/): provider event verification.
- [Safaricom Daraja](https://developer.safaricom.co.ke/): direct M-Pesa integration entry point, not a verified tariff quotation.
- [OWASP object authorization](https://owasp.github.io/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/): authorization review framework.
- [ODPC legislation](https://www.odpc.go.ke/data-protection-laws-kenya/): local privacy review reference; legal applicability and registration decisions need qualified review.

## Subsequent developer clarification

The developer reports the original build price as **KES 30,000** and confirms that conversion tracking is configured in the Google Ads account. These are user-provided commercial/account facts, not independently verified invoice or account observations. E13 remains a code finding: base Ads tag present, explicit Ads conversion-event dispatch and GA4 ecommerce purchase implementation not found. Account-side URL rules/imports/external tags were not inspected; zero recorded conversions is not established.

## Small-business context supplied for commercial revision
Zentora operates a physical shop and an online store approximately one month old. Reported early estimates are five online orders/month, KES 50,000 online sales/month and KES 10,000 average order value. Physical-shop revenue/profit, online margins, traffic and hosting invoices remain unknown. The KES 30,000 historical build and friendship-based commercial relationship do not establish future hours/rates or available development cash. The owner may struggle to fund even a KES 50,000 lump sum.

Provisional affordability targets supplied by the user: KES 3,000–8,000 critical corrections, KES 5,000–12,000 marketing/storefront, KES 8,000–20,000 essential order/payment improvements; advanced work separate. Revised C/M/O/P/B packages explicitly narrow scope, estimate effort independently of discounted commercial prices and identify work that exceeds these targets. Business figures/targets are not independently verified invoices, margins or approved quotations. Technical source findings are preserved; no new production investigation or functional testing was performed for this commercial revision.

## Checks executed
- Passed: `go test ./internal/domain/discovery ./internal/service/discovery ./internal/service/worker ./internal/handlers/discovery ./internal/repository/postgres`. Executed with GOPROXY=off, GOTOOLCHAIN=local and a temporary Go cache; all five packages returned `ok`. Existing tests use isolated fixtures/mocks for these paths; no application server or production database was started.
- Passed: frontend application TypeScript `tsc --noEmit --incremental false -p D:/Projects/ZentoraUI/tsconfig.app.json` using the installed local compiler. This is a type check, not a browser test or Vite production build.
- Backend working tree initially clean; final changes are only the new audit-reports directory. Frontend git status was clean using a command-scoped safe.directory option; no global git configuration changed.
- No frontend end-to-end runner, core payment/checkout transaction test suite, production load test or current vulnerability scan was executed. Passing discovery/type checks does not establish production readiness.

## Required verification handover
Obtain deployed commit IDs, sanitized schema-only export plus migration ledger, provider dashboards/screenshots without customer records, backup restore evidence, monitoring and billing summaries, current campaign conversion definitions and tag inventory. Run isolated two-customer authorization tests, reservation concurrency tests and provider sandbox scenarios. Browser checks must cover 360/390/768/1440px, keyboard checkout, labels/focus/contrast, slow-network errors, feed-selected variants, crawler HTML, redirects, consent and duplicate events. Run dependency/SBOM scans and record advisories, reachability and remediation; this report does not assert a vulnerability-free build.
