# Zentora budget and cost model — incremental small-business plan
Revised 22 September 2026. All prices are **provisional planning ranges, not accepted quotations**. This revision replaces the earlier enterprise-scale budget as the current investment proposal. It retains the full functional vision in the feature inventory but does not price every future feature as an immediate purchase.

## 1. Business baseline
Zentora has a physical shop and a roughly one-month-old online store. User-supplied estimates: **5 online orders/month, KES 50,000 online sales/month, KES 10,000 average order value**. These are not verified accounting records. Physical-shop sales/profit, online margins, traffic, catalog size, Ads spend and current infrastructure invoices remain unknown.

The original **KES 30,000** website price is historical context, not an outstanding balance or a formula for future charges. A KES 50,000 payment may be unaffordable. Monthly sales are not available development cash: stock, fulfillment, rent, transport, marketing and other costs must be funded first.

## 2. Effort and commercial price are different
Estimates below are task-based ranges based on the existing audit, not newly measured implementation results. Hours include focused investigation of the affected path, coding, tests, one review/release cycle and short handover. They assume familiar code, a compatible deployed schema, one storefront, KES, one selected payment provider, and prompt access/decisions. They exclude external approval waiting time.

Prices are proposed discounted relationship-based planning allowances. **They do not represent market rates, a confirmed offer or a reduction in technical effort.** For example C1's 8–14 hours at KES 3,000–5,000 implies about KES 214–625/hour across the range extremes. That is a substantial commercial concession, not evidence that the fix takes little work. Whoever undertakes a package must explicitly accept its scope/price; otherwise reduce the scope safely or obtain a revised estimate. Do not remove tests to force a price.

No fixed hourly rate has been agreed. Consequently hours are estimated independently; multiplying them by an invented rate would give false precision. A formal quote can instead use agreed hours × agreed rate minus an explicitly agreed discount. Stage boundaries do not mean every item must be bought.

## 3. Small-package ledger
| Package | Estimated hands-on hours | Provisional KES |
|---|---:|---:|
| C1: Protect customer order details | 8–14 | 3,000–5,000 |
| C2: Correct Merchant stock reporting | 6–10 | 2,000–3,000 |
| C3: Close confirmed permission and credential-log defects | 10–18 | 4,000–8,000 |
| C4: Check recovery and current deployment | 6–12 | 3,000–6,000 |
| M1: Essential Ads conversion and GA4 measurement | 20–32 | 5,000–9,000 |
| M2: Small homepage and promotion refresh | 6–10 | 2,000–3,000 |
| O1: Make checkout and stock changes reliable | 28–44 | 8,000–12,000 |
| O2: Simple COD/payment collection register | 12–20 | 4,000–8,000 |
| P1: M-Pesa integration build and sandbox verification | 40–64 | 15,000–25,000 |
| P2: M-Pesa exception handling and controlled release | 16–28 | 6,000–10,000 |
| B1: One demonstrated operations bottleneck | 8–16 | 3,000–6,000 |

The scope/benefit/reuse/dependencies/acceptance/exclusions for each package follow below. Shared foundations are purchased once: P1/P2 assume O1/O2 rather than charging again to rebuild them. M1 does not include P1/P2. B1 selects one outcome, not all operations features.

### Fit against the requested affordability targets
| Target | Feasible initial scope | Assessment |
|---|---|---|
| Critical corrections KES 3,000–8,000 | C1 alone KES 3,000–5,000; C1+C2 KES 5,000–8,000 | Useful first correction, **not full security/stabilization** |
| Marketing/storefront KES 5,000–12,000 | M1 alone KES 5,000–9,000; M1+M2 KES 7,000–12,000 | Limited measurement and presentation; no marketing CMS |
| Essential payments/orders KES 8,000–20,000 | O1 alone KES 8,000–12,000; O1+O2 KES 12,000–20,000 | Reliable existing checkout plus manual collection records; **no automatic M-Pesa** |
| Advanced capabilities | Individually scoped later | No initial commitment or artificial full-platform total |

Other confirmed security/recovery issues still require C3/C4 or immediate safe containment. C1+C2+C3+C4 together are **30–54 hours / KES 12,000–22,000**; this explicitly exceeds the narrow corrective target. O1 addresses additional order/financial integrity defects and may need to precede marketing. Calling C1+C2 “complete stabilization” would be misleading.

A minimally reliable automated M-Pesa release needs both P1 and P2: **56–92 hours / KES 21,000–35,000**, **after O1/O2**. Including those foundations, the payment/order path is **96–156 hours / KES 33,000–55,000**, excluding earlier C1–C4 and supplier charges. This exceeds the proposed KES 8,000–20,000 target: split funding, keep existing verified manual/COD operations, or defer automation. P1 is independently reviewable sandbox work, but cannot be safely launched without P2. This is still a narrow service with manual exception/refund operation, not every production-grade finance feature.

## 4. Package definitions
### C1 — Protect customer order details
**Business objective:** Prevent one customer from viewing another customer's orders.

- **Exact deliverables:** Bind customer order list/detail queries to the signed-in account; keep a separate authorized admin path; regression tests for missing/changed filters and another customer's order.
- **Reuse:** Existing authentication, order handlers, services and queries (T01/E03).
- **Expected benefit:** Protect delivery/contact details and customer trust.
- **Effort:** 8–14 hours, including focused testing.
- **Provisional commercial price:** KES 3,000–5,000.
- **Dependencies:** Confirm deployed revision and have a safe release/rollback route. No live customer-data probing.
- **Acceptance:** Two isolated customer accounts cannot list/read each other's orders; ordinary customers cannot access guest orders; intended admin access still works.
- **Excluded:** Full security audit, admin MFA, guest tracking links and unrelated account features.

### C2 — Correct Merchant stock reporting
**Business objective:** Stop available products being incorrectly marked unavailable in Shopping.

- **Exact deliverables:** Align feed eligibility and availability helpers with current reserve semantics; fixture checks before/after reservation and release; validate one regenerated feed.
- **Reuse:** Current XML feed, stable productID-variantID identifiers and inventory SQL (T06/E06/E11).
- **Expected benefit:** More accurate product availability; no promise of more sales.
- **Effort:** 6–10 hours, including focused testing.
- **Provisional commercial price:** KES 2,000–3,000.
- **Dependencies:** Confirm which inventory quantity is free to sell. C2 does not fix the separate cancellation lifecycle.
- **Acceptance:** Five units minus three reserved leaves two sellable in feed/page/checkout fixtures; IDs and variant URLs stay unchanged.
- **Excluded:** Full Merchant diagnostics dashboard, feed caching/freshness jobs, stock reconciliation and order-linked reservation release.

### C3 — Close confirmed permission and credential-log defects
**Business objective:** Ensure denied permissions stay denied and credentials do not enter request logs.

- **Exact deliverables:** Repair SQL grant/revoke grouping; stop ordinary query-token authentication; redact sensitive query parameters; handle affected socket authentication safely; targeted permission/logging tests.
- **Reuse:** Existing role/session middleware and logger (T09/T10/E21/E30).
- **Expected benefit:** Reduce confirmed access-control and credential-exposure risks.
- **Effort:** 10–18 hours, including focused testing.
- **Provisional commercial price:** KES 4,000–8,000.
- **Dependencies:** Identify clients depending on query tokens; retain a safe admin route; assess whether existing logs require controlled investigation/credential rotation.
- **Acceptance:** Explicit revokes override role grants; inactive permissions deny; synthetic credentials never appear in logs; required socket access works safely or is disabled.
- **Excluded:** Forensic investigation, full history secret scanning, broad session redesign, every T11 hardening item and external penetration testing.

### C4 — Check recovery and current deployment
**Business objective:** Know where the store runs and be able to recover its data.

- **Exact deliverables:** Document actual host/DB/media ownership and existing invoices; verify Redis settings against the running arrangement; create or confirm encrypted backup; perform one isolated restore; add one failure alert and short rollback instructions.
- **Reuse:** Existing Render/Netlify/Compose configuration, database tools and current host capabilities (T12/E17–E19).
- **Expected benefit:** Reduce avoidable downtime/data loss without an automatic hosting upgrade.
- **Effort:** 6–12 hours, including focused testing.
- **Provisional commercial price:** KES 3,000–6,000.
- **Dependencies:** Authorized infrastructure access and an isolated restore destination; backup storage charges separately confirmed.
- **Acceptance:** Owner knows account ownership; backup restores representative schema/data safely; alert reaches responsible operator; deployment settings match application expectations.
- **Excluded:** Managed 24/7 response, migration-system overhaul, hosting migration, complex disaster recovery and repair of previously unknown infrastructure faults.

### M1 — Essential Ads conversion and GA4 measurement
**Business objective:** Understand visits, shopping activity and submitted orders without calling unpaid orders paid sales.

- **Exact deliverables:** Inspect existing Ads action source/trigger; implement one agreed order-placement conversion with server-returned order ID and authoritative merchandise value; explicit once-per-order dispatch and reload/retry checks; GA4 page/product/add-to-cart/checkout/order-placement events; consent accept/reject/withdraw controls; simple funnel view and handover.
- **Reuse:** Base Ads tag, existing order response, product/feed IDs and SPA routes (T08/E13).
- **Expected benefit:** Evidence for Ads and storefront decisions; order-placement measurement clearly labelled separately from collected sales.
- **Effort:** 20–32 hours, including focused testing.
- **Provisional commercial price:** KES 5,000–9,000.
- **Dependencies:** C1/C2 and relevant C3 controls; Google account access; authoritative order values. If totals remain unsafe, use count-only order placement with no claimed sales value until O1. Primary paid-purchase conversion waits for verified collection/payment.
- **Acceptance:** Inspect actual action before creating any duplicate; one intended event per controlled order despite reload; failed submit produces no conversion; rejected/withdrawn consent blocks optional tags; item IDs match feed; no PII in ordinary events; diagnostics and account receipt checked.
- **Excluded:** Online payment integration, automatic paid COD purchase/refund dispatch, BigQuery, enhanced conversions, sophisticated attribution, cross-device tracking and advanced marketing dashboards.

### M2 — Small homepage and promotion refresh
**Business objective:** Present current offers and featured products clearly.

- **Exact deliverables:** Update up to three existing hero slides and one existing featured selection; correct links, offer dates and delivery wording; mobile layout check.
- **Reuse:** Existing hero components, images, catalog and discovery feeds (T14/E25).
- **Expected benefit:** Clearer, current shop presentation.
- **Effort:** 6–10 hours, including focused testing.
- **Provisional commercial price:** KES 2,000–3,000.
- **Dependencies:** Owner supplies approved images, copy, products and truthful prices. Do not advertise checkout discounts until supported.
- **Acceptance:** Approved content displays on mobile/desktop, links reach correct products and no unsupported discount/delivery promise remains.
- **Excluded:** Self-service banner CMS, new photography, campaign scheduler, coupon engine, blog and ongoing content changes.

### O1 — Make checkout and stock changes reliable
**Business objective:** Prevent duplicate orders and release held stock correctly when an order is cancelled.

- **Exact deliverables:** Request idempotency with database constraint; atomic cart conversion; order-line stock allocation and controlled cancel/fulfill actions; exact monetary calculation and sellability check; explicit final price/charge policy. Where discount/delivery automation is out of scope, remove unsupported promotion claims and require explicit approval of any later delivery quote before fulfilment/collection.
- **Reuse:** Current transactional order creation, atomic reserve/release, carts and status admin (T02–T04/E04–E06).
- **Expected benefit:** Fewer duplicate orders, trapped stock and customer price disputes.
- **Effort:** 28–44 hours, including focused testing.
- **Provisional commercial price:** KES 8,000–12,000.
- **Dependencies:** C1/C4; reconcile existing reservations and agree cancellation/delivery policy. This integrity work moves ahead of M1 monetary conversions where needed.
- **Acceptance:** Concurrent retry creates one order; cancellation releases each allocation once; fulfilled stock cannot be casually cancelled; last-unit contention stays nonnegative; changed prices/extra delivery charges cannot silently become accepted totals.
- **Excluded:** Full coupon/tax engine, automatic multi-location optimization, courier quotes, online payments and automated refunds. Historical stock repairs beyond a bounded opening reconciliation require re-scope.

### O2 — Simple COD/payment collection register
**Business objective:** Separate an order being placed from money actually received.

- **Exact deliverables:** Persist payment method; restricted collection recording with reference, amount, date and actor; basic order/payment/fulfillment distinction; collection export/checklist and mismatch list for the small order volume.
- **Reuse:** Existing order/payment table scaffolding and admin order screens (T05/T13/E07/E29).
- **Expected benefit:** Five monthly orders can be checked against actual cash or provider statements.
- **Effort:** 12–20 hours, including focused testing.
- **Provisional commercial price:** KES 4,000–8,000.
- **Dependencies:** O1; approved collection/refund responsibilities; manual entry must be checked against receipt/statement, not a customer's screenshot alone.
- **Acceptance:** An unpaid order cannot become paid from a general status change; recorded collection reconciles to evidence; duplicates are rejected; sales view separates placed from collected.
- **Excluded:** Automated M-Pesa initiation/callbacks, refund automation, accounting system, settlement imports and automatic conversion dispatch.

### P1 — M-Pesa integration build and sandbox verification
**Business objective:** Build one reliable M-Pesa payment path, initially kept out of live checkout.

- **Exact deliverables:** One eligible provider adapter; intent/attempt/reference records; provider-specific callback authenticity/correlation; verified amount/currency/order matching; durable event deduplication; pending/status query and retry UI; sandbox success/decline/replay tests.
- **Reuse:** Existing order UI and payment scaffolding plus O1/O2 foundations (T05).
- **Expected benefit:** A reviewed sandbox payment flow ready for operational validation.
- **Effort:** 40–64 hours, including focused testing.
- **Provisional commercial price:** KES 15,000–25,000.
- **Dependencies:** O1/O2/C4, eligible merchant account and documented provider contract. Direct Daraja versus gateway chosen after account/fee review, not assumed.
- **Acceptance:** Sandbox callbacks cannot pay the wrong order or amount; duplicate callbacks/retries do not double-collect; secrets stay server-side; P1 completion alone does not enable live payments.
- **Excluded:** P2 outage/late-success operational acceptance, second provider, cards, automated partial refunds, subscriptions and seller payouts.

### P2 — M-Pesa exception handling and controlled release
**Business objective:** Handle uncertain payments safely before accepting live automated payments.

- **Exact deliverables:** Missing/delayed/out-of-order event and late-success-after-cancel tests; persistent retry/reconciliation path; exception list; statement-matching runbook; restricted manual refund/reversal procedure through provider; rollout/rollback, alert and owner handover.
- **Reuse:** P1 records and O1/O2 controls.
- **Expected benefit:** A narrowly scoped live payment service with an accountable daily exception process.
- **Effort:** 16–28 hours, including focused testing.
- **Provisional commercial price:** KES 6,000–10,000.
- **Dependencies:** P1; provider access, statements and refund authority; acceptable backup/monitoring. Any live test payment requires separate explicit authorization.
- **Acceptance:** Timeout remains unknown until verified; late success is reallocated or escalated/refunded; no duplicate collection; daily totals match statements; rollback preserves payment history. Do not release while these checks fail.
- **Excluded:** Unattended accounting reconciliation, comprehensive refund UI, automated provider failover, high-availability redesign, sophisticated fraud tooling.

### B1 — One demonstrated operations bottleneck
**Business objective:** Remove one recurring manual problem the owner identifies.

- **Exact deliverables:** Select exactly one: connect customer order-confirmation email with a bounded persistent retry path; OR add a simple approved delivery-rate rule to the existing checkout; OR an auditable stock-adjustment reason/history screen. Re-scope if the chosen variant exceeds the allowance.
- **Reuse:** Existing SMTP templates, shipping tables or inventory admin, depending on selection.
- **Expected benefit:** Less repetitive administration where the business actually needs it.
- **Effort:** 8–16 hours, including focused testing.
- **Provisional commercial price:** KES 3,000–6,000.
- **Dependencies:** O1 and relevant source review/acceptance detail for the chosen variant; provider charges separate.
- **Acceptance:** Demonstrate the selected operation and its failure path: mail outage retains pending delivery; delivery rule produces agreed totals; or stock adjustment preserves actor/reason and nonnegative quantities.
- **Excluded:** All three options together, support ticketing platform, broad fulfillment/returns system, SMS, courier API and every future operations capability.

## 5. Incremental approval, payments and uncertainty
Agree each package in writing before work: exact deliverables, price, acceptance demonstration, access and rollback responsibility. One optional payment arrangement is 30% at start, 40% at staging demonstration and 30% after acceptance. For a KES 5,000 package that is KES 1,500 / 2,000 / 1,500. Instalments spread timing; they do not reduce the total or imply credit terms have been accepted.

Do not charge for shared work twice if packages overlap. Scope changes, previously unknown schema defects, difficult historical-stock cleanup or unsuitable provider accounts trigger a stop/re-estimate before extra spending. As a cash-planning example, reserve 15% above a selected package if affordable: C1+C2 KES 5,000–8,000 becomes KES 5,750–9,200 available funds. That reserve is **not** included in the package price or an automatic extra charge. If unaffordable, choose C1 first; maintain containment on unresolved defects.

Testing and handover are included in hours; provider fees, new infrastructure charges, taxes if applicable, Ads media spend, production customer transactions, catalog preparation and legal/accounting services are not. Confirm invoice/tax treatment when agreeing a price.

## 6. Current-scale monthly operating budget
**Actual present expenditure must be confirmed before calculating the incremental cost of new features.** Five orders do not determine traffic, availability or database/storage requirements. Existing hosting may be adequate; the audit does not demonstrate capacity or durability.

### What existing configuration establishes
- Frontend netlify.toml describes a Netlify deployment and proxy to a Render API hostname. It proves intended routing, not the paid plan or live bill.
- render.yaml declares free API and Redis plans. Do not assume those are deployed, still eligible or suitable for persistent/security data solely from the file.
- Compose is an alternative API+Redis arrangement with an external PostgreSQL dependency; it is not proof that a separate server is billed alongside Render.
- PostgreSQL is external to the supplied deployment manifest; its host, invoice, backup retention and restore capability are unknown.
- ImageKit integration and local media fallback exist. Verify durable local uploads if fallback is used; a free application filesystem must not be assumed persistent.
- Redis settings differ between manifest and application; revocation state shares an eviction policy intended for cache. These are correctness/security checks, not reasons to automatically buy a bigger server.
- There is no justification at this scale for Kubernetes, a separate search engine, paid analytics warehouse or additional replicas without measured need.

### Current baseline and incremental allowances
| Expense | Current amount | Proposed treatment / incremental monthly KES |
|---|---|---|
| Frontend/API hosting | Confirm invoices and actual topology | Retain if safe; no automatic new subscription |
| PostgreSQL and Redis | Confirm provider, plan and retention | Retain if secure/durable; upgrades only against a documented failed requirement |
| Media/CDN | Confirm usage and provider bill | Retain existing arrangement; avoid paying twice for bundled storage/transfer |
| Domain/DNS | Confirm annual renewal invoice | Annual renewal / 12; retain existing domain |
| Essential backup/monitoring capacity | Check what existing plans include | **0–1,000** allowance if existing capabilities or modest extra storage suffice |
| Transactional email | Existing provider/limits unknown | **0–300** allowance for the small volume, subject to provider terms |
| Optional SMS | No new service assumed | **0** by default; optional 10 messages × illustrative KES 2 = KES 20, plus any sender/setup/minimum fees |
| Payment transaction fees | Selected merchant agreement unknown | Fee rate × successfully collected volume by method; no fee assumed on an unpaid order |
| Optional maintenance | No retainer assumed | **0** mandatory; optional KES 1,000–3,000 for narrowly agreed checks/support time, not unlimited fixes |
| Google Ads | Confirm actual spend | Separate owner-controlled budget; no automatic increase |
| Physical shop expenses | Unknown | Inventory, rent, staff, transport, utilities and fulfillment remain separate |

The backup/email amounts are planning allowances, not supplier quotes. If the existing plan cannot meet backup/recovery/security needs, obtain the real incremental quote; do not pretend KES 1,000 covers any required upgrade. Configuration correction/recovery setup labor is C3/C4, not a recurring subscription.

**Illustrative incremental software overhead:** existing hosting retained + included backups/email = **KES 0** new mandatory subscription; modest backup/monitoring and email additions = **up to KES 1,300/month** under the stated allowance. These are **incremental**, not the total hosting bill and not a guarantee of safe free hosting. Actual total = existing confirmed recurring bills + approved additions + transaction fees. Optional maintenance/Ads/SMS are separate.

Illustrative payment-fee sensitivity for KES 50,000 successfully collected online volume: 1% = KES 500, 1.5% = KES 750, 2% = KES 1,000. These are hypothetical rates, not verified current tariffs. If only half the volume uses that provider, halve the percentage fee. COD, failed payments, refunds, settlement transfers, taxes and minimum charges depend on contract; do not apply a gateway percentage to all reported sales automatically.

Domain illustration only: if annual renewal is KES 2,000, its monthly equivalent is about KES 167, not a new monthly charge. Replace with the actual invoice. The former KES 45,550 low-volume allowance is retired from the current-scale proposal.

## 7. Affordability based on contribution, not sales
“Contribution” below means sales less product cost and direct selling expenses; it may still have to cover shared rent, wages, tax and other overhead. These are **hypothetical scenarios, not Zentora's margins or available cash**.

| Assumed contribution margin on KES 50,000 sales | Monthly contribution | Illustrative 20% allocation to improvements |
|---|---:|---:|
| 10% | 5,000 | 1,000 |
| 20% | 10,000 | 2,000 |
| 30% | 15,000 | 3,000 |

Only make an allocation after necessary overhead, inventory replenishment and cash commitments are protected. Physical shop profit cannot be assumed to fund the website because it is unknown. At KES 1,000 / 2,000 / 3,000 saved monthly, a KES 5,000 package takes approximately **5 / 3 / 2 months**, assuming no withdrawals. Urgent vulnerabilities need containment immediately, not several months of open exposure while saving.

No sales uplift, payback or ROAS is promised. With five monthly orders, percentages swing substantially with one order. Review customer activity, submitted/collected orders, cancellations, Ads spend and manual workload over several weeks; do not infer causation from tiny samples.

## 8. Later capabilities and re-quotation
Delivery automation, returns/refunds UI, customer support, coupons, campaign scheduling, bulk catalog/procurement, loyalty, retention automation, profitability and optional sellers remain in the full feature matrix. No whole-platform or seller-expansion price is recommended now. Quote a bounded capability when its trigger occurs; provider onboarding, legal/payment liability and seller settlement require separate discovery. The architecture proposal is a design direction, not approval to purchase every component.
