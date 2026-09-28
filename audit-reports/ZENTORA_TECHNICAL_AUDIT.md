# Zentora technical audit
22 September 2026. Read with [evidence register](ZENTORA_AUDIT_EVIDENCE.md), whose E references resolve to source paths and lines. This is a source audit and production verification plan, not a penetration-test certificate.

## Commercial revision note
The owner-facing report, budget, roadmap and feature matrix now use a small-business incremental plan based on approximately five monthly online orders. Technical findings T01–T15 and their evidence remain unchanged. W01–W18 references below are retained long-term scope identifiers, not current priced packages; the matrix maps their bounded C/M/O/P/B slices and deferred work. Former enterprise phase budgets and whole-scope effort allocations are superseded. Unfunded security/financial defects require the roadmap's safe containment, not indefinite acceptance of risk.

## Architecture and reusable implementation
Go/Gin modular monolith with handler/service/repository layers, pgx PostgreSQL access, Redis caching/session support, SMTP, ImageKit/local image fallback and WebSockets. React/Vite SPA uses React Router, TanStack Query, Axios and Zustand. Catalog, variant, category closure, attributes, inventory locations, cart, wishlist, reviews, discovery and administration are substantial reusable components (E01–E02, E14–E15, E24–E28). Keep this architecture.

The baseline schema already contains order_payments, shipping_methods/rates, fulfillment/tracking and banners. These are not fully connected workflows. The much larger merged_schema.sql/missing.sql are alternative schema assets with schema namespaces, seller/payment/consent entities and TimescaleDB assumptions. Applying them wholesale would be an uncontrolled migration, not feature activation (E16–E17).

## Prioritized findings
Each item includes source evidence, impact, remediation and a verifiable release gate. Severity expresses potential business impact; production exploitability was not tested.

### T01 — Critical: order object authorization
**Broken, E03.** Customer order list/detail routes authenticate but do not bind the database query to the current identity. List permits caller-supplied user_id or no user filter; detail takes a raw order ID. The repository returns shipping contact/address fields. This creates a cross-customer data exposure path.
**Remediation:** separate customer/admin query contracts; require identity in customer service/repository predicates, reject guest order access except scoped expiring opaque tracking credentials. Never rely on frontend filters.
**Gate:** two isolated test customers cannot list/read each other's orders by any filter; guest orders stay inaccessible; admins retain explicitly authorized access. Add regression tests before deployment.

### T02 — High: incomplete stock/order lifecycle
**Partial/broken, E04–E06.** Reserve atomically decrements available and increments reserved, a useful overselling safeguard. Order status update does not release cancellation stock or consume reservations on fulfillment, does not validate old-to-new transitions, and lacks reservation-to-order/location records. First-location allocation may reject a variant stocked elsewhere.
**Remediation:** order-line allocation and reservation records; expire/release/consume exactly once inside transaction; state transition matrix with expected version; movement ledger with actor and reason.
**Gate:** last-unit concurrent checkouts produce one success; repeated cancel/expiry leaves stock unchanged after first release; dispatched stock cannot be cancelled without returns workflow; multi-location allocation totals agree.

### T03 — High: customer total, feed price and charged amount can diverge
**Partial, E04, E07, E12, E26.** Order uses current variant price and ignores discount resolution, tax and shipping; UI shows estimated subtotal. Cart accepts client price snapshots, while final order correctly recomputes base price. Feed has separate sale-price logic. Money calculations use float64 despite DECIMAL storage. Order does not check product publication status in the item validation block.
**Remediation:** one server quote/pricing contract for listing, cart, checkout, feed and payment; fixed-point minor units or decimal arithmetic; approved tax policy; product sellability checks; quote expiry and explicit price-change acceptance.
**Gate:** active sale, expiry, stale/tampered cart, inactive product, shipping zone and rounding scenarios agree at every surface. Complete delivery price is shown before commitment.

### T04 — High: duplicate order creation
**Partial, E05.** No request idempotency in connected creation flow. Cart cleanup/conversion happens after commit with errors ignored. Browser retries or parallel requests may create multiple orders.
**Remediation:** idempotency key plus request hash/unique constraint, transactionally convert the cart, return prior result on identical retry and conflict on changed payload.
**Gate:** parallel retries return one order and one reservation set; simulated response loss safely retries.

### T05 — High: no connected online payment or reconciliation workflow
**Missing integration with schema scaffolding, E07.** M-Pesa disabled; payment method ignored; no routed payment webhook/refund processing found. COD is an order-placement option, not proof of cash collection.
**Remediation:** implement proposed payment state machine, one provider first, COD collection ledger, reconciliation and refund approvals.
**Gate:** duplicate, delayed, forged/mismatched, missing and out-of-order provider events cannot incorrectly mark paid or ship; amount/currency/reference must match; refunds cannot exceed captured amount.

### T06 — High: Merchant stock semantics contradict checkout
**Broken, E06, E11.** After reserving 3 of 5 units, available=2 and reserved=3. Feed calculates 2-3 and advertises unavailable despite 2 sellable units.
**Remediation:** define available as free-to-sell consistently; update feed eligibility and availability helpers; reconcile existing stock before rollout.
**Gate:** shared stock fixtures agree across feed, product page and checkout before/after reserve, cancel and fulfill. Preserve current productID-variantID identifiers.

### T07 — Medium: feed regeneration and incomplete incremental freshness
**Partial, E09–E12.** Full feed rebuild precedes conditional response; timestamp ETag changes between runs. Incremental gate overlooks inventory-only updates, existing variant-price changes unless product timestamp is touched, and promotion expiry.
**Remediation:** scheduled versioned full feed snapshot, content hash, last-success time/item counts; explicit dirty variants or change sequence if incremental publishing is needed; test deletions and sale transitions. Retain last good feed on failures.
**Gate:** unchanged snapshot returns stable ETag without full DB work; inventory/price-only updates and expiry appear within agreed freshness target (initially 30 minutes, validated against scale).

### T08 — High: Ads measurement and consent gap
**Account/code distinction:** the developer reports conversion tracking configured in Google Ads. The reviewed codebase has the base Ads tag but no explicit Ads conversion-event dispatch or GA4 purchase-event implementation. The account was not inspected; URL-based, imported or externally managed conversions may exist. Do not interpret this as proof that the account records zero conversions. Validate its actual action source and trigger before implementing or duplicating conversion actions.

**Ads configured; GA4 workflow missing, E13/E33.** Base Ads script runs at page load. No connected consent gate, ecommerce purchase event, GA4 property configuration or attribution capture found. External account-side tags remain unverified.
**Remediation:** owner-owned GA4/GTM/Ads accounts, controlled tag inventory, consent interface and defaults, shared item-ID contract, server-backed conversion dispatch and COD conversion policy. See architecture.
**Gate:** consent decline/withdrawal controls nonessential tags; one pageview per SPA route; no PII in URL/events; one canonical purchase conversion per order; distinguish placed, collected and refunded.

### T09 — High: query tokens enter request logs
**Source-confirmed path, E21.** Auth accepts query bearer tokens and logging records raw query. WebSocket token extraction also allows query tokens.
**Remediation:** remove ordinary query auth; redact sensitive query parameters immediately; use short-lived one-use socket tickets if needed; establish restricted retention and investigate historical exposure without copying secrets.
**Gate:** synthetic tokens never appear in logs/referrers; ordinary APIs reject query credentials; socket origin/auth tests pass.

### T10 — High: permission revoke precedence
**Broken query logic, E30.** SQL AND binds more tightly than OR, so role-derived grants can bypass the user-specific revoke and active-permission checks that follow the second grant branch.
**Remediation:** group grant alternatives before applying exclusions; define explicit deny precedence; invalidate sessions/claims on permission changes.
**Gate:** role grant plus explicit revoke denies access, inactive permissions deny, and expected direct grants still work.

### T11 — Medium: API/browser hardening
**Partial, E20–E23.** Bcrypt, login/reset/verification throttling and admin role checks exist; do not replace them. Wildcard origin with credentials is an invalid credentialed-browser CORS combination, not proof of bypass. Response helper includes raw error strings. HTTP server has no explicit timeout fields. Bearer storage is localStorage.
**Remediation:** explicit origin policy, sanitized public errors with request IDs, read-header/body/idle limits, upload byte/pixel limits and robust content validation; endpoint abuse limits for guest orders, search events and feed generation. Review CSP and token/session design. Cookie migration, if chosen, must include HttpOnly/Secure/SameSite and CSRF controls.
**Gate:** malicious origin, oversized payload/image, slow request, malformed input and XSS fixtures fail safely; no DB details leak. Verify limits at both proxy and app. No confirmed SQL injection/XSS exploit is claimed.

### T12 — High: deployment and recovery not reproducible from supplied configuration
**Configured/unverified, E17–E19, E31.** Compose has API and Redis but no worker/PostgreSQL; external DB is plausible. Render defines REDIS_URL while app reads REDIS_ADDR/PASS. Separate worker sends order reminders but is not deployed by these manifests; metrics also run inside API. CI and migration scripts are empty. Redis allkeys-lru also holds security blacklist keys; evaluate revocation behavior under eviction. Render Redis IP allowlist permits all sources in source configuration; actual deployment is unknown.
**Remediation:** choose authoritative topology, managed private DB/Redis access, separate security state from evictable cache or enforce durable revocation; one scheduler/leased jobs; migration ledger, CI, staging, backup/PITR, restore and rollback runbooks. Explicitly build/deploy worker when needed.
**Gate:** clean staging deploy, schema migration rehearsal, Redis outage/eviction revocation tests, single job ownership under two replicas, timed restore and rollback. Initial proposed RPO 1 hour/RTO 4 hours, subject to owner cost approval.

### T13 — Medium: notifications and business dashboards overstate completion
**Partial, E08/E29.** Customer order emails are skipped in connected service paths; admin mail is best effort. Dashboard sums completed/delivered order totals as revenue, independent of payment/refunds and order completion time.
**Remediation:** durable outbox, retry/dead-letter queue, real customer contact snapshots, delivery receipts; paid/collected/refunded ledgers and explicit metric definitions.
**Gate:** committed order queues one message; provider outage retries without duplicates; customer receives correct status. Dashboard reconciles to payment/settlement ledger and dates.

### T14 — Medium: CMS and back-office workflow gaps
**Partial, E24–E25.** Product/discount/inventory/order admin exists; active hero content is a source constant. Homepage service is reusable but not exposed as a complete marketing editor. No connected returns, supplier purchasing, campaign scheduling, support ticket or consent-preference admin workflow found.
**Remediation:** extend admin with draft/preview/publish/schedule/archive, validated media/link targets, scoped roles and audit trails; business staff manage shipping/returns/campaigns.
**Gate:** staff complete a campaign and order exception without code or SQL, changes are attributable and reversible.

### T15 — High assurance gap: insufficient core-commerce tests
**Partial, E32.** Existing discovery tests are useful. No checkout/payment/inventory concurrency or end-to-end customer tests were found; frontend package has no test script. No claim that dependencies are safe is justified from manifest inspection.
**Remediation:** isolated database integration suite, authorization regression tests, provider sandbox tests, browser journeys, dependency scanning and staging load test.
**Gate:** critical cases listed above pass in CI; load target and latency budgets are measured before capacity promises.

## Customer journey assessment
| Stage | Current source behavior | Required outcome / acceptance | Work |
|---|---|---|---|
| Land | Hero, navigation, public policies; Ads tag | Content truthful, consent before optional tags, crawlable metadata | W03/W11/W12 |
| Discover/search | PostgreSQL fuzzy/full-text feeds, filters, suggestions, search events | Relevance fixtures, useful zero-results, bounded event abuse | W12/W14 |
| Product | Variants, images, stock, wishlist, JSON-LD; feed variant URL supported | Same variant/price/availability in feed, page and quote | W03/W04 |
| Cart | Local guest vs server signed-in cart | Defined merge on login, stale-price notice, quantity validation | W04 |
| Checkout | Guest/address-book flow, subtotal estimate | Valid Kenyan address, full delivery quote, idempotent submit | W04/W06 |
| Payment | COD option, M-Pesa disabled | Verified online payments; retry/pending/timeout UI; explicit COD collection | W05 |
| Confirmation | Success UI, skipped customer email path | Persisted confirmation, opaque guest tracking, retryable notification | W06/W08 |
| Track/deliver | Account orders/admin status; tracking schema | Owner-scoped tracking, shipment events, failed delivery/redelivery, proof | W01/W07 |
| Return/refund/support | Informational pages, schema proposals | Order-linked request, approval, partial refund, restock decision | W07/W08 |

W references in this journey table describe the original full functional scope. Current small-package estimates and exclusions are mapped in the revised feature matrix and budget; do not assume a small package completes an entire W group. Abandoned checkout must not hold stock forever; payment-unknown orders require query/reconciliation before cancellation; paid-but-unavailable exceptions require refund/escalation rather than silent substitution.

## Database review and migration risks
Retain product/variant/attribute/category closure separation. Keep immutable order line descriptions and financial snapshots even when catalog changes. Current order creation populates name/slug/prices but leaves some SKU/image snapshot fields unset. Add location allocations, reservations and stock movement history, not duplicate inventory counters. Validate nonnegative balances, positive quantities, unique provider references/idempotency keys, currency consistency and constrained status changes in both service and database. Add pending-work composite indexes based on EXPLAIN in staging; existing user/order/status and discovery indexes are a starting point. Never claim a migration applied from its presence in git.

## UX, SEO and live-review limits
Source has responsive Tailwind layouts, mobile filters/cart controls, loading placeholders, lazy route loading, error boundaries, SEO components and crawler edge functions. These are implementation assets, not measured accessibility/performance scores. Live extraction was limited (E28/E33 and public observations). Manually verify screen readers/keyboard, input labels and errors, image layout shifts, mobile checkout, variant canonical URLs, robots/sitemap freshness and edge/bot output parity. Avoid a frontend rewrite; consider selective prerendering only if measured crawl/index coverage warrants it.
