# Zentora feature gap matrix
Revised 22 September 2026. Full single-vendor commerce vision, Google Merchant Center/Shopping, Google Ads and optional future independent sellers. Current business estimates are approximately 5 online orders / KES 50,000 sales per month after one month online; physical-shop profit and available improvement funds are unknown.

Status refers to source, not proven production service. “Configured/unverified” includes schema-only assets. “Missing” means no connected implementation found in reviewed routes/services/repositories/UI; it does not rule out manual operations or external tools. [Evidence register](ZENTORA_AUDIT_EVIDENCE.md) defines E references.

**This is a complete capability inventory, not an immediate shopping list.** W01–W18 are retained scope identifiers for cross-references in the technical audit; their former whole-package prices/effort are superseded. Current purchasable slices are C1–C4, M1/M2, O1/O2, P1/P2 and one selected B1 outcome in the [budget](ZENTORA_BUDGET_AND_COST_MODEL.md). A small package does not complete every capability in a W group. The staged mapping below identifies the remainder as separately scoped future work.

Stages: S1 urgent fixes/containment; S2 essential measurement/presentation; S3 reliable orders and optional automatic payments; S4 demand-led operations; S5 optional growth; S6 optional sellers. An unresolved security or financial-integrity problem must be fixed or safely restricted immediately even if its broader feature group is later. Current commercial priority is C1 first, C2 if affordable, then remaining urgent controls; do not treat sales as available profit.

### Ads account versus website implementation

The developer reports conversion tracking configured in Google Ads. In the reviewed codebase, only the base Ads tag was found; explicit conversion-event dispatch and GA4 purchase tracking were not found. Account-side URL rules, imports or external tags remain unverified. The gap is end-to-end website conversion integration and validation, not a claim that no account conversion action exists.

## A. Catalog and merchandising
Evidence: **E15/E24/E27/E28**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Scope / indicative stage |
|---|---|---|
| Product CRUD, publication/archive, brands, nested categories, tags and attributes | Implemented | W04/W13; S3/S5 |
| Variants, SKU, image galleries and specifications | Implemented | W01/W04; S1/S3 |
| Bulk import/export and bulk editing | Missing | W13; S5 |
| Publication scheduling/moderation, video, comparison and bundles | Partial | W11/W13; S2/S5/S5 |
| Related products, trending, best sellers and recommendations | Implemented | W12/W14; S2/S5/S5 |
| Discounts, historical pricing and tax | Partial | W04/W11; S3/S2/S5 |
| Ratings and verified-purchase reviews | Implemented | W07/W12; S4/S2/S5 |
| Product questions/answers, cross-sell and upsell | Partial | W13; S5 |

- **Product CRUD, publication/archive, brands, nested categories, tags and attributes:** Validate public visibility and safe archival; reuse admin/catalog.
- **Variants, SKU, image galleries and specifications:** SKU uniqueness, snapshot completion, image safety and content quality gates.
- **Bulk import/export and bulk editing:** Dry-run validation, queued import/export and row error report.
- **Publication scheduling/moderation, video, comparison and bundles:** Publication fields/admin exist; scheduling/moderation/video/comparison/bundles not found as connected workflows.
- **Related products, trending, best sellers and recommendations:** Evaluate relevance, event coverage and cold starts before external engine.
- **Discounts, historical pricing and tax:** Discount CRUD/targets exist; final checkout ignores them; approve tax and snapshot prices.
- **Ratings and verified-purchase reviews:** Ownership and completed-order check exist; verify UI submission, moderation and completion timestamp.
- **Product questions/answers, cross-sell and upsell:** Related feeds reused; Q&A and explicit upsell flows not found.

## B. Inventory
Evidence: **E04–E06/E11/E15**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Stock quantities, locations, adjustments and stock indicators | Implemented | W03/W04; S1/S3 |
| Reservations and prevention of overselling | Partial | W04; S3 |
| Release on cancel/failure/expiry; consume on fulfillment | Broken | W04; S3 |
| Movement audit and reconciliation | Partial | W04/W07; S3/S4 |
| Low-stock, back-in-stock alerts, returns restocking | Missing | W07/W08; S4 |
| Suppliers, purchase orders and warehouse procurement | Missing | W13; S5 |
| Turnover and inventory reporting | Partial | W14; S5 |

- **Stock quantities, locations, adjustments and stock indicators:** Reuse admin and atomic updates; align meaning across feed/page/order.
- **Reservations and prevention of overselling:** Atomic reserve exists; add per-order allocation, expiry and concurrency acceptance.
- **Release on cancel/failure/expiry; consume on fulfillment:** No order lifecycle release/consume; enforce exactly-once movements.
- **Movement audit and reconciliation:** Quantity changes exist; add immutable movement history and drift report.
- **Low-stock, back-in-stock alerts, returns restocking:** Threshold jobs, consented subscriptions and inspected-return disposition.
- **Suppliers, purchase orders and warehouse procurement:** Existing locations reused; purchasing, receiving and supplier reconciliation workflow.
- **Turnover and inventory reporting:** Current counts/metrics insufficient for stock valuation; add cost/movement views.

## C. Accounts
Evidence: **E20/E24/E26/E33**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Registration/login/logout/password recovery/email OTP | Implemented | W01/W08; S1/S4 |
| Phone verification/admin MFA | Missing | W01/W08; S1/S4 |
| Profiles, address book, order history and wishlist | Partial | W01/W06; S1/S3 |
| Guest checkout and saved carts | Implemented | W04/W06; S3 |
| Preferences, support history, privacy export/deletion | Missing | W08; S4 |
| Session management and admin permissions | Partial | W01; S1 |

- **Registration/login/logout/password recovery/email OTP:** Verify email templates, throttling and UI completion; preserve bcrypt.
- **Phone verification/admin MFA:** Provider/OTP abuse controls; prioritize admin MFA over customer friction.
- **Profiles, address book, order history and wishlist:** Connected UI/APIs, but order history authorization is broken.
- **Guest checkout and saved carts:** Define login cart merge/repricing and guest order tracking.
- **Preferences, support history, privacy export/deletion:** Policies/schema proposals are not self-service controls; retention-aware workflows.
- **Session management and admin permissions:** Existing sessions/roles; query logging, revoke precedence and stale claim checks.

## D. Cart and checkout
Evidence: **E04/E05/E07/E26**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Guest/server cart, quantity updates and availability | Partial | W04; S3 |
| Cart synchronization and duplicate order prevention | Partial | W04; S3 |
| Server repricing, coupons and final delivery/tax total | Partial | W04/W06/W11; S3/S2/S5 |
| Mobile progress, address validation and confirmation | Partial | W06/W12; S3/S2/S5 |
| Abandoned cart/checkout recovery | Missing | W11; S2/S5 |

- **Guest/server cart, quantity updates and availability:** Server-cart source exists; resolve price-at-added trust and enforce final quote.
- **Cart synchronization and duplicate order prevention:** Separate carts; define merge and atomic conversion/idempotency.
- **Server repricing, coupons and final delivery/tax total:** Base variant repriced; coupon/shipping/tax not applied end-to-end.
- **Mobile progress, address validation and confirmation:** Responsive checkout exists; authoritative errors/total and recovery needed.
- **Abandoned cart/checkout recovery:** Consent and contact capture, no false purchase, reservation expiry independent of marketing.

## E. Payments
Evidence: **E07/E16/E29**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| COD orders and cash collection | Partial | W05; S3 |
| M-Pesa STK and hosted cards | Missing | W05; S3 |
| Payment intents/attempts/provider references/idempotency | Configured/unverified | W05; S3 |
| Callback authentication, duplicate/delayed/out-of-order events | Missing | W05; S3 |
| Timeout/query, provider outage and failed-payment recovery | Missing | W05; S3 |
| Manual Till/Paybill matching, under/overpayment | Missing | W05/W07; S3/S4 |
| Refunds/reversals/partial refunds/settlement/fees | Configured/unverified | W05/W07; S3/S4 |
| Exact currency/money, finance audit and reports | Partial | W04/W05/W10; S3/S2/S5 |

- **COD orders and cash collection:** COD choice exists; method discarded and collection/reconciliation ledger absent.
- **M-Pesa STK and hosted cards:** One provider adapter and safe hosted/tokenized card flow.
- **Payment intents/attempts/provider references/idempotency:** DDL scaffolding/proposals exist; wire stateful service and constraints.
- **Callback authentication, duplicate/delayed/out-of-order events:** Durable inbox, provider-specific validation, amount/currency/reference checks.
- **Timeout/query, provider outage and failed-payment recovery:** Pending/unknown UI, status query, retry attempts and late-success exception.
- **Manual Till/Paybill matching, under/overpayment:** Receipt uniqueness, exception queue and dual-control corrections.
- **Refunds/reversals/partial refunds/settlement/fees:** Expanded SQL is not operation; refund caps and reconciled ledger.
- **Exact currency/money, finance audit and reports:** KES/DECIMAL present; replace float arithmetic and status-based revenue.

## F. Orders
Evidence: **E03–E08/E15/E29**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Creation, unique order number and historical lines | Implemented | W04; S3 |
| Payment/fulfillment/delivery state separation | Partial | W04/W07; S3/S4 |
| Customer/admin cancellation, partial fulfillment and order notes | Partial | W07; S4 |
| Return/exchange/refund workflows | Missing | W07; S4 |
| Invoices/receipts, notifications and financial reconciliation | Partial | W06/W07/W08; S3/S4 |
| Fraud review and operational audit | Missing | W07; S4 |

- **Creation, unique order number and historical lines:** Complete SKU/image snapshots and idempotency; retain order history.
- **Payment/fulfillment/delivery state separation:** Tables exist; current admin edits one order status without transition rules.
- **Customer/admin cancellation, partial fulfillment and order notes:** Admin cancel status only; add guarded commands, allocations and history.
- **Return/exchange/refund workflows:** Request/approval/inspection/refund/restock; exchange creates linked replacement.
- **Invoices/receipts, notifications and financial reconciliation:** Templates/stats exist; durable customer delivery and collected-value documents needed.
- **Fraud review and operational audit:** Velocity/exception flags and authorized review queue; do not auto-reject by unexplained scoring.

## G. Delivery
Evidence: **E07/E15/E24**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Kenyan addresses, Nairobi/nationwide zones, rate rules | Partial | W06; S3 |
| Delivery estimates and pickup | Configured/unverified | W06/W07; S3/S4 |
| Courier handoff, tracking, fulfillment dashboard | Partial | W07; S4 |
| Failed delivery/redelivery/proof/returns logistics | Missing | W07/W08; S4 |
| Delivery performance reporting | Missing | W10; S2/S5 |

- **Kenyan addresses, Nairobi/nationwide zones, rate rules:** Address fields and rate tables exist; implement quoted coverage/location/weight rules.
- **Delivery estimates and pickup:** No complete checkout selection/ETA flow found; define pickup locations and promises.
- **Courier handoff, tracking, fulfillment dashboard:** Admin status and schema exist; manual shipment workflow first, API optional.
- **Failed delivery/redelivery/proof/returns logistics:** Event/reason history, customer notifications and proof references.
- **Delivery performance reporting:** Dispatch/delivery timestamps, on-time denominator and failure/retry cost.

## H. Back office
Evidence: **E17/E24/E25/E29/E30**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Catalog, categories, inventory, discounts and order admin | Implemented | W01/W07; S1/S4 |
| Payments/refunds/customer/support/shipping configuration | Partial | W07/W08; S4 |
| Role permissions and sensitive action audit | Partial | W01/W07; S1/S4 |
| Dashboard, exports and operational reporting | Partial | W10; S2/S5 |
| Banners, campaigns, suppliers and system configuration | Partial | W11/W13; S2/S5/S5 |

- **Catalog, categories, inventory, discounts and order admin:** Extend existing forms/tables; scoped actions and safe validation.
- **Payments/refunds/customer/support/shipping configuration:** User APIs and order admin exist; end-to-end business operator flows incomplete.
- **Role permissions and sensitive action audit:** Role guards exist; repair deny precedence, scoped permissions, audit refund/stock/content.
- **Dashboard, exports and operational reporting:** Current counts/revenue source insufficient for collected sales and margins.
- **Banners, campaigns, suppliers and system configuration:** Discount UI present; homepage hard-coded and supplier workflow absent.

## I. Marketing and content
Evidence: **E12/E13/E15/E25**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Hero/carousel, category banners and featured placements | Partial | W11; S2/S5 |
| Seasonal/flash sales/scheduled promotions | Partial | W04/W11; S3/S2/S5 |
| Coupons/vouchers: percent/fixed/minimum/usage/customer/free shipping | Partial | W11; S2/S5 |
| Bundles/cross-sell/upsell/personalized offers | Partial | W13/W14; S5 |
| Referral/loyalty/gift cards | Missing | W14; S5 |
| Newsletter, email/SMS campaigns and segmentation | Missing | W08/W11; S4/S2/S5 |
| Landing pages/blog/buying guides/social sharing | Partial | W11/W12; S2/S5 |
| Ads attribution, reporting and consent-aware remarketing | Partial | W03/W10; S1/S2/S5 |
| Merchant feed and diagnostics | Partial | W03/W10; S1/S2/S5 |

- **Hero/carousel, category banners and featured placements:** Hero constants plus section models; staff editor, preview, scheduling and links.
- **Seasonal/flash sales/scheduled promotions:** Discount windows/targets exist; unify checkout application and CMS activation.
- **Coupons/vouchers: percent/fixed/minimum/usage/customer/free shipping:** Discount types exist; coupon redemption limits and checkout policy incomplete.
- **Bundles/cross-sell/upsell/personalized offers:** Related/recommendation feeds exist; commercial bundling and targeted offers optional.
- **Referral/loyalty/gift cards:** Optional balances/rules/liability controls; select scope before building.
- **Newsletter, email/SMS campaigns and segmentation:** Opt-in/audience/suppression/delivery reporting; transactional mail is separate.
- **Landing pages/blog/buying guides/social sharing:** Static public/SEO assets; editable campaign landing content, optional editorial publishing.
- **Ads attribution, reporting and consent-aware remarketing:** Base Ads tag exists; GA4/dispatch/consent and account links need validation.
- **Merchant feed and diagnostics:** Connected variant XML feed; stock/freshness/price mismatches and account diagnostics.

## J. Search/discovery
Evidence: **E14/E32**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Full-text/typo-tolerant search, autocomplete/suggestions | Implemented | W12/W14; S2/S5/S5 |
| Category/brand/price/attribute filters and sorting | Implemented | W12; S2/S5 |
| Trending/best sellers/related/recommended/recent views | Partial | W12/W14; S2/S5/S5 |
| Search analytics/zero-results/relevance/performance | Partial | W10/W12; S2/S5 |

- **Full-text/typo-tolerant search, autocomplete/suggestions:** Existing PostgreSQL FTS/trigram implementation; tune with query fixtures.
- **Category/brand/price/attribute filters and sorting:** Verify frontend/backend contracts and explain plans on realistic scale.
- **Trending/best sellers/related/recommended/recent views:** Multiple candidate feeds exist; event quality and recent-view UI need verification.
- **Search analytics/zero-results/relevance/performance:** Search events/clicks exist; operator reporting and abuse/privacy controls needed.

## K. Communications
Evidence: **E08/E20/E31**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Transactional email templates and authentication emails | Implemented | W08; S4 |
| Order/payment/shipment/refund notifications | Partial | W06/W08; S3/S4 |
| SMS, preferences, templates, retry/delivery tracking | Partial | W08; S4 |

- **Transactional email templates and authentication emails:** SMTP source; delivery/domain DNS operationally unverified.
- **Order/payment/shipment/refund notifications:** Customer confirmation skipped; admin notification and reminder source present.
- **SMS, preferences, templates, retry/delivery tracking:** Notification models/WebSockets; SMS/outbox/preferences not complete.

## L. Service and trust
Evidence: **E27/E33**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Contact/help/returns/privacy/terms pages | Implemented | W08/W12; S4/S2/S5 |
| Tickets, order-linked complaints/escalation and dispute history | Missing | W08; S4 |
| Authenticity/quality controls, verified reviews and trust | Partial | W07/W12; S4/S2/S5 |

- **Contact/help/returns/privacy/terms pages:** Source pages; validate promises, local requirements and actual operations.
- **Tickets, order-linked complaints/escalation and dispute history:** Support case UI/admin, SLAs, attachments and audit.
- **Authenticity/quality controls, verified reviews and trust:** Review checks exist; inspect product sourcing and truthful delivery/payment promises.

## M. Analytics and business intelligence
Evidence: **E13/E14/E29**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| GA4, ecommerce funnel, cart abandonment and product performance | Missing | W03/W10; S1/S2/S5 |
| Acquisition/retention/AOV/repeat rate/campaign ROAS | Partial | W10; S2/S5 |
| GMV vs revenue, payment/refund/cancellation/delivery metrics | Partial | W05/W07/W10; S3/S2/S5 |
| Inventory turnover, LTV and contribution margin | Missing | W14; S5 |
| Operational dashboards and exports | Partial | W10; S2/S5 |

- **GA4, ecommerce funnel, cart abandonment and product performance:** Typed events, consent, single SPA views, canonical purchases and DebugView QA.
- **Acquisition/retention/AOV/repeat rate/campaign ROAS:** Ads base only; paid/collected cohort dashboards and attribution coverage.
- **GMV vs revenue, payment/refund/cancellation/delivery metrics:** Order status aggregates not financial truth; payment/fulfillment reporting views.
- **Inventory turnover, LTV and contribution margin:** Cost snapshots and cohort model; show unknowns instead of fabricated profit.
- **Operational dashboards and exports:** Existing admin dashboard extend with filters/access control/CSV redaction.

## N. Security and operations
Evidence: **E03/E17–E23/E30–E32**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Object authorization and permission revoke precedence | Broken | W01; S1 |
| Passwords, auth throttling, JWT and sessions | Implemented | W01; S1 |
| SQL injection/XSS/CSRF/CORS/file upload/API abuse | Partial | W01; S1 |
| Secrets, error exposure, logs and dependency assurance | Partial | W01/W02; S1 |
| Backups/restore/DR/rollback/environment separation | Configured/unverified | W02/W09; S1/S4 |
| CI/CD/migrations/background jobs/alerts | Partial | W02/W08/W09; S1/S4 |
| HTTPS/headers/pooling/indexes/capacity | Partial | W02/W12/W14; S1/S5 |
| Automated tests/accessibility/performance/SEO | Partial | W01/W09/W12; S1/S2/S5 |

- **Object authorization and permission revoke precedence:** Critical customer order scoping and deny-precedence fixes.
- **Passwords, auth throttling, JWT and sessions:** Retain controls; test revoke/eviction, role changes and admin MFA.
- **SQL injection/XSS/CSRF/CORS/file upload/API abuse:** Parameterized query examples exist; broader attack tests unperformed; harden limits/policy.
- **Secrets, error exposure, logs and dependency assurance:** Do not leak tokens/raw errors; scan history/dependencies in controlled CI.
- **Backups/restore/DR/rollback/environment separation:** No operational proof; establish runbooks and measured recovery.
- **CI/CD/migrations/background jobs/alerts:** Empty pipeline/migrate placeholders, separate worker not in manifests.
- **HTTPS/headers/pooling/indexes/capacity:** TLS DB/pool/indexes exist; public headers/capacity and topology require verification.
- **Automated tests/accessibility/performance/SEO:** Discovery tests/SEO code exist; core transactions/browser/load checks incomplete.

## O. Optional marketplace
Evidence: **E16**. Feature-specific exceptions/defects are detailed in the technical audit.

| Capability | Current status | Work / target |
|---|---|---|
| Seller onboarding/verification/storefronts/roles/moderation | Configured/unverified | W15/W18; S6 |
| Seller offers/pricing/inventory/multi-seller carts/order splits | Configured/unverified | W16; S6 |
| Commissions/balances/settlements/payouts/refund allocation | Configured/unverified | W17; S6 |
| Seller fulfillment/performance/dashboard/disputes/support | Configured/unverified | W15/W16/W18; S6 |
| Kubernetes/microservices/card storage | N/A | None; Deferred |

- **Seller onboarding/verification/storefronts/roles/moderation:** Expanded schema exists, no connected seller product; optional implementation.
- **Seller offers/pricing/inventory/multi-seller carts/order splits:** Reuse catalog and stock; add seller ownership and immutable allocations.
- **Commissions/balances/settlements/payouts/refund allocation:** Implement balanced ledger, approvals, reconciliation and payout-provider agreement.
- **Seller fulfillment/performance/dashboard/disputes/support:** Seller and platform operations plus controlled pilot.
- **Kubernetes/microservices/card storage:** No current requirement; hosted/tokenized cards and modular monolith preferred.

## Revised scope-to-package mapping
The row stages above describe broad destinations; this mapping controls **what is included now**. All remaining functionality is still documented, not deleted or assumed delivered.

| Retained scope ID | Current bounded slice | Remainder / decision |
|---|---|---|
| W01 security | C1 customer order scoping; C3 permission/log fixes | Other confirmed exposures contained now; full hardening/MFA separately scoped, not included |
| W02 infrastructure | C4 deployment/restore check | Full CI/migration/recovery programme separately scoped where evidence requires it |
| W03 Merchant/consent | C2 stock semantics; M1 consent and measurement | Feed snapshots/diagnostics/freshness automation separately quoted |
| W04 checkout/inventory | O1 duplicate prevention, stock lifecycle and price integrity | Full promotions/tax engine, allocation optimization deferred S5; unsafe totals restricted now |
| W05 payments | O2 manual collection; P1+P2 one-provider M-Pesa after foundations | Cards, second provider, full refund automation/settlement imports separate |
| W06 delivery/confirmation | B1 only if that one delivery/email outcome is selected | General delivery configuration/tracking separately scoped S4 |
| W07 order operations | O1 guarded changes; O2 collected-payment view | Partial fulfillments, returns/exchange/support UI separately scoped S4; controlled manual process now |
| W08 communications/support | B1 only if confirmation/retry is selected | SMS, support tickets/privacy self-service/general outbox separate S4; privacy requests handled manually now |
| W09 release/migration | Package-specific testing/release and C4 restore | No broad historical migration included; extra remediation quoted before work |
| W10 measurement | M1 essential funnel/order-placement tracking | Collected-purchase/refund dispatch, full attribution and profit dashboards separate after reliable finance data |
| W11 campaigns | M2 limited existing homepage/featured updates | CMS/coupons/abandonment/scheduling deferred S5; no self-service editor included in M2 |
| W12 UX/SEO | M2 limited mobile/content/link check | Wider accessibility/SEO work scoped to demonstrated gaps; no full certification included |
| W13 advanced catalog | None in initial purchase | S5 bulk import, procurement, bundles when repeated work/demand justifies it |
| W14 retention/scaling | None in initial purchase | S5 loyalty, cohort profitability, measured capacity work only on evidence |
| W15–W18 sellers | None in initial purchase | S6 separate commercial/operating decision, discovery and quote |

## Current effort and price register
Ranges are estimates, not guaranteed delivery quotations; hours include focused testing and are not reduced because the price is discounted.

| Package | Estimated hours | Provisional KES |
|---|---:|---:|
| C1 | 8–14 | 3,000–5,000 |
| C2 | 6–10 | 2,000–3,000 |
| C3 | 10–18 | 4,000–8,000 |
| C4 | 6–12 | 3,000–6,000 |
| M1 | 20–32 | 5,000–9,000 |
| M2 | 6–10 | 2,000–3,000 |
| O1 | 28–44 | 8,000–12,000 |
| O2 | 12–20 | 4,000–8,000 |
| P1 | 40–64 | 15,000–25,000 |
| P2 | 16–28 | 6,000–10,000 |
| B1 | 8–16 | 3,000–6,000 |

Do not multiply a package price by the number of feature rows that reference it. Dependencies, exact deliverables, exclusions and failure-case acceptance live in the [budget](ZENTORA_BUDGET_AND_COST_MODEL.md); timing/containment gates live in the [roadmap](ZENTORA_IMPLEMENTATION_ROADMAP.md). Long-term unselected capability effort is **not yet estimated under the lean scope**; quote after requirements are bounded, instead of carrying over the old enterprise person-day allocations.

Shared dependencies remain unchanged: authorization before customer order self-service; reliable price/stock before online payment; payment truth before automated refund/settlement/paid conversion; consent before optional tracking/remarketing; costs before profitability; financial allocations before seller settlement. M1 measures submitted COD orders explicitly, not confirmed payment. Automatic M-Pesa requires both P1 and P2 after O1/O2; partial funding is not a release gate.

At approximately five monthly orders, manually checked collection, fulfillment, support and refund records may be appropriate. Manual processes must be access-controlled, auditable and reconcile to real receipts/stock. Scale alone does not justify insecure access, incorrect payment confirmation or silent price changes.
