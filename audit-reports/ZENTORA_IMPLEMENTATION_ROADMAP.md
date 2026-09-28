# Zentora incremental implementation roadmap
Revised 22 September 2026. This supersedes the former P0–P5 enterprise delivery schedule. Current context: physical shop plus approximately one month online, 5 orders/month and KES 50,000 monthly online sales, all user-provided estimates. Original KES 30,000 build cost is historical. No application changes are made by this audit revision.

## Delivery approach
Purchase one useful outcome at a time. Package hours estimate real work including focused testing; price ranges are provisional discounted commercial allowances, not confirmed offers or market-rate calculations. Exact package contracts are in the [budget](ZENTORA_BUDGET_AND_COST_MODEL.md). No full-time team or uninterrupted development schedule is assumed.

The existing catalog, admin, stock SQL, order flow, feed, frontend and database remain the foundation. There is no immediate CMS rebuild, paid warehouse or new infrastructure tier. Existing security and financial defects require correction or safe containment regardless of order volume.

## Stage 1 — Immediate corrections and containment
**First choice:** C1 alone (8–14 hours, KES 3,000–5,000). Add C2 (6–10 hours, KES 2,000–3,000) if affordable. Together: **14–24 hours / KES 5,000–8,000**.
**Other urgent work:** C3 (10–18 hours, KES 4,000–8,000) and C4 (6–12 hours, KES 3,000–6,000). C1–C4 totals **30–54 hours / KES 12,000–22,000**, not a KES 3,000–8,000 full stabilization promise.

- Protect customer order list/detail paths and retain authorized admin operations (T01).
- Correct Merchant free-to-sell quantity calculations without changing product IDs (T06).
- Repair permission deny precedence and prevent credential leakage through query authentication/logs (T09/T10).
- Confirm actual deployment/settings, account ownership, recoverable data/media backup and one operational alert (T12).

**Gate:** isolated two-customer tests, stock fixtures, permission/log redaction tests and restore evidence. Each package has its own acceptance; completion of C1 does not clear T09/T10.
**Owner decision:** fund the next urgent correction or keep affected paths safely disabled. Review whether existing deployed data/logs require an incident assessment; do not claim a breach without evidence.
**Release dependency:** at least a protected backup and rollback path before code deployment; full C4 assessment need not block emergency endpoint containment.

### Safe handling until fixes are available
- **C1:** If the fix cannot be released promptly, temporarily disable affected customer order-read endpoints at the server. Staff can handle enquiries through a separately protected admin route.
- **C2:** If stock cannot be trusted, exclude affected items from advertised inventory until checked; do not advertise an invented quantity.
- **C3:** Restrict affected administrative actions; stop sensitive query logging and disable query-auth/socket paths where a safe replacement is not ready. Disabling a UI button alone is insufficient.
- **C4:** Until a backup is verified, take a protected export before any release and avoid risky migrations; restrict exposed data services immediately if confirmed.
- **Order/financial integrity (O1):** restrict risky status edits; use a controlled order-to-stock reconciliation and recorded adjustments. Remove unsupported discount claims and clearly obtain acceptance of any separately quoted delivery fee before fulfillment/collection. If safe handling cannot be guaranteed, pause order submission. Do not silently overcharge or blindly release reservation quantities without evidence.
- **Consent/measurement:** if current optional advertising tags cannot honor the approved consent policy, temporarily pause them. Do not interpret unverified conversion reporting as collected revenue.

Containment is a temporary restriction requiring authorized operational action, not something this report has implemented. It may reduce customer convenience; it must actually block the unsafe server behavior, not merely hide a button. Review unresolved issues before each release.

## Stage 2 — Essential marketing and storefront improvements
**M1:** 20–32 hours / KES 5,000–9,000. **M2:** 6–10 hours / KES 2,000–3,000.
Combined **26–42 hours / KES 7,000–12,000**; either can be commissioned alone.

M1 checks the existing Ads action before adding code. It connects one explicitly named order-placement event, a basic GA4 shopping funnel and consent controls. Reuse feed productID-variantID identifiers. Stable order IDs and controlled dispatch/reload checks prevent duplicate conversion reporting. COD order placement is not a paid purchase. Confirmed-payment/refund automation is excluded until reliable collection/payment records exist. No ordinary analytics event carries customer contact details.

M2 refreshes up to three existing hero slides and one existing featured selection with owner-supplied assets. No elaborate marketing CMS, automated coupons or content scheduler is commissioned.

**Dependencies:** Stage 1 access/consent controls; verified order totals before sending conversion values. O1 moves ahead of monetary measurement where needed. A count-only submitted-order event may be used temporarily if it is accurately labelled and safe; do not invent a monetary value.
**Gate:** diagnostics/account receipt verified, one intended event per successful controlled order, failed/repeated submission does not inflate results, reject/withdraw consent works, IDs match feed, agreed content/links/mobile views correct.
**Owner decision:** review actual acquisition and submitted-versus-collected results over several weeks before increasing Ads spend. At five orders/month, avoid drawing strong causal conclusions.

## Stage 3 — Orders and payments in separately funded steps
**O1:** 28–44 hours / KES 8,000–12,000. **O2:** 12–20 hours / KES 4,000–8,000.
Combined **40–64 hours / KES 12,000–20,000**.

O1 prevents duplicate orders, records stock allocations, makes cancellation/fulfillment changes safe, and ensures price handling is explicit. It reuses current stock transactions and order snapshots. It does not add a full coupon/tax/shipping engine.
O2 persists payment method and adds restricted evidence-backed collection recording, a basic mismatch list and export. At current volume a checked register plus provider statements is practical; automated settlement imports are unnecessary.

**Gate:** one order under concurrent retry; last-unit sale safe; cancellation releases once; no invisible extra charges; order status alone cannot mark paid; duplicate collection references rejected; collections matched against independent receipt/statement evidence.
**Sequencing exception:** these correctness fixes are urgent if the current workflow cannot be safely controlled, so do not delay them merely because the stage is numbered after marketing.

### Optional automatic M-Pesa once foundations are ready
**P1 sandbox build:** 40–64 hours / KES 15,000–25,000.
**P2 operational checks/release:** 16–28 hours / KES 6,000–10,000.
Together **56–92 hours / KES 21,000–35,000**, additional to O1/O2. Total O1/O2/P1/P2: **96–156 hours / KES 33,000–55,000**, excluding Stage 1 and provider fees. This exceeds the suggested payment affordability target; defer automation or spread funding rather than remove safeguards.

Choose direct M-Pesa or a gateway after checking the actual merchant account, provider capabilities, fees and operating responsibilities. A provider is not selected merely to match the price target. Keep cards and a second provider outside this scope.

P1 delivers a sandbox-tested one-provider flow. P2 proves amount/order/currency matching, duplicate/missing/delayed/out-of-order callback handling, durable reconciliation/retry, late-payment exceptions, manual provider refund/reversal procedure and owner handover. **P1 alone is not permission to switch on live payments.** No timeout is assumed unpaid or paid without verification.

**Gate:** all payment scenarios in T05 pass; statements reconcile; unknown payments have an owner and process; alerts/backups/rollback work. Any live test transaction is separately authorized.
**Owner decision:** enable only once both stages and provider approval pass. Continue verified COD/manual merchant collection when not funded or ready. No automatic refunds, sophisticated fraud platform or unattended accounting is promised.

## Stage 4 — Business operations driven by recurring work
**B1 allowance:** one selected outcome, 8–16 hours / KES 3,000–6,000. Choose email confirmation with bounded persistent retry, one approved delivery-rate rule, or stock adjustment reason/history. It is not a bundle of all three.

Reuse current email templates, delivery tables or admin inventory screen as appropriate. Confirm the exact variant and its failure test before quoting. If infrastructure/schema conditions make that variant larger, re-scope rather than reduce reliability.

Keep low-volume courier handoff, support and returns in an organized manual process: order reference, responsible person, delivery evidence, customer consent to changes, refund evidence and inspection/restocking decision. Never treat a manual process as nonexistent simply because a ticketing or courier API is absent.
**Gate:** selected operation works and fails safely; staff can perform it without direct SQL.
**Owner decision:** approve additional operations UI only when repeated work/errors justify it. There is no predetermined monthly purchasing sequence.

## Stage 5 — Growth features when justified
Retain the long-term scope for campaigns/coupons, banner self-service, bulk catalog, procurement, automated abandoned-cart messages, loyalty/referrals, search tuning, profitability dashboards and more integrations. **Separate quote at decision time; no current price or engineering effort commitment.**

Suggested triggers:
- Repeated homepage/campaign changes cause delays: quote a small banner editor before a full CMS.
- Repeated delivery/payment mismatches: prioritize operational reconciliation, not another marketing feature.
- Catalog updates exceed practical manual effort: assess bulk import with dry-run/error handling.
- Repeat customers request rewards and unit economics can support them: define a bounded loyalty policy.
- Several weeks of reliable funnel/collection records reveal an actual loss point: improve that step first.
- Measured performance or provider limits fail acceptable service: tune current queries/caching before buying new platforms.

**Gate:** define outcome, cost, privacy/security controls and a way to measure success before approval. Cost snapshots must exist before claiming profit/LTV. No guaranteed uplift/payback.

## Stage 6 — Optional marketplace
Independent sellers, verification/storefronts, seller offers/inventory, split orders, commissions, settlement/payouts, disputes and seller reporting remain a future business choice. No marketplace development is in the initial investment.
**Prerequisites:** stable core stock/payment/refund records, seller demand, operating staff, liability/policy review and an eligible payout arrangement.
**Gate:** separately scoped discovery and quotation, then seller isolation and balanced-ledger tests before pilot. The existing Google Merchant Center feed is not third-party seller infrastructure.

## Estimated effort versus elapsed time
No firm delivery date is assumed. At an illustrative 10 focused implementation hours/week:
- C1 alone: roughly 1–2 working weeks; C1+C2 roughly 2–3.
- M1+M2: roughly 3–5 weeks.
- O1+O2: roughly 4–7 weeks.
- P1+P2: roughly 6–10 weeks after foundations, plus merchant onboarding/approval waits.

Actual availability may differ. Critical containment cannot wait for these schedules; arrange prompt authorized action. Hours include package testing, but external account access, approvals and merchant onboarding can extend the calendar. Funding pauses between independently useful packages are expected; payment foundations are not bypassed during a pause.

## Shared release checks
Keep the verified T01–T15 findings and acceptance criteria in the technical audit. Test only in isolated fixtures/provider sandbox until a reviewed release is authorized. Backup before changes; demonstrate the package's adverse paths, obtain acceptance, release with rollback, and check the outcome. Existing discovery/type checks are not substitutes for authorization, order-concurrency or payment tests.

## Traceability to prior technical scope
W01–W18 still appear in the preserved technical/architecture documents as long-term scope identifiers, not purchasable packages or current prices. The matrix maps them to C/M/O/P/B packages and deferred stages. Former P0–P5 enterprise budgets/timelines are superseded by this six-stage plan; do not combine them.
