# Zentora: online shop assessment and gradual improvement plan
**Prepared for the shop owner · Revised 22 September 2026 · All amounts in KES**

## 1. Executive summary
Zentora already has a useful online shop alongside its physical store. The next step is to make the existing shop safer, measure what customers do and improve the parts that cause real problems. A large rebuild or a large upfront software commitment is not recommended.

Start with a small correction: protect customer order details, then correct product availability sent to Google Shopping. Keep unresolved security and order problems safely restricted until they are fixed. Add better advertising measurement before spending more on advanced marketing. Introduce automatic M-Pesa only when both payment confirmation and exception handling can be tested properly.

## 2. Current business position
The website cost **KES 30,000** and has been operating for approximately **one month**. Current estimates are **five online orders and KES 50,000 in online sales per month**, averaging about **KES 10,000 per order**. These are early business estimates, not verified accounts. Physical-shop sales and profit are unknown.

The original website price is historical expenditure, not an outstanding balance. Monthly sales are also not available spending money: stock, rent, transport, delivery, marketing and other costs come first. Improvements should be approved one at a time from an affordable business budget.

## 3. What the online shop already offers
- Products, categories, brands, variants, images, search and product recommendations.
- Customer accounts, addresses, wishlists, guest/customer carts and order placement.
- Product, stock, discount and order administration.
- A Google Merchant Center product feed and a Google Ads base tag.
- Public contact, help and policy pages.

These are existing capabilities found in the code. Some journeys remain incomplete, and their operation in the live store has not all been verified. Reuse the current website rather than paying to rebuild these features.

## 4. Important issues
| Issue | What it means for the shop |
|---|---|
| **Customer order privacy** | The code does not properly restrict some order reads to the customer who owns them. Correct this urgently; disable the affected customer order views at the server if a safe fix cannot be released promptly. |
| **Incorrect Google Shopping stock** | The feed subtracts held stock twice, so sellable products can appear unavailable. Correct the calculation while retaining product identifiers. |
| **Other confirmed access/logging defects** | Some denied permissions may remain effective, and credentials can enter request logs. Restrict affected access and stop sensitive logging until corrected. |
| **Checkout and stock handling** | Retries can create duplicate orders; cancelled orders do not automatically return reserved products to available stock. Discounts and delivery charges are not fully applied in the order total. |
| **Incomplete payment process** | M-Pesa is disabled in checkout. An order or a “completed” status is not evidence that money has been received. |
| **Advertising measurement gap** | Google Ads conversion tracking is reportedly configured in the account, but the reviewed website code has only the base tag, with no explicit conversion-event or GA4 purchase implementation. |
| **Recovery and communication gaps** | Backups and the actual hosting setup need confirmation; customer order-email delivery is incomplete. |

The Ads account was not inspected: it may have URL-based or imported conversions. The finding does **not** prove that it records zero conversions. It means accurate website-to-account measurement still needs checking and completion. No actual customer-data breach was established by this source review.

## 5. Recommended improvements and priorities
**Immediately:** protect customer information, correct Merchant stock reporting and contain other confirmed access risks. Verify a recoverable backup before changes. Where stock or prices cannot be handled safely, restrict the affected order action or pause online order submission rather than accepting unreliable orders.

**Next, in affordable steps:** fix duplicate orders and cancellation stock where needed before monetary conversion reporting. Connect the existing Ads action correctly, add essential GA4 measurement and consent choices, and refresh a small number of homepage offers.

**After those foundations:** keep a simple collection register for cash/manual payments. Build automatic M-Pesa only when the shop can fund verified payment handling and the process for delayed, failed or uncertain payments.

**As recurring needs arise:** improve delivery rules, customer confirmations, stock adjustments and support. Advanced marketing and third-party sellers can wait.

## 6. Affordable phased development budget
These are **provisional discounted planning ranges**, not agreed quotations. They assume reuse of the current website and limited, defined outcomes. Lower commercial prices do not remove the work or testing needed. Approve each package separately.

| Improvement | What the owner receives | Estimated KES |
|---|---|---:|
| C1: Customer order privacy | Customers can access only their own orders, with targeted checks | **3,000–5,000** |
| C2: Merchant stock correction | Accurate availability calculation in the existing feed | **2,000–3,000** |
| C3: Permission/logging corrections | Confirmed permission and credential-log defects corrected | **4,000–8,000** |
| C4: Recovery and deployment check | Confirmed hosting ownership, one restore check and basic failure alert | **3,000–6,000** |
| M1: Essential Ads and GA4 | Tested order-placement measurement, a basic shopping funnel and consent controls | **5,000–9,000** |
| M2: Homepage refresh | Up to three existing slides and one featured selection updated | **2,000–3,000** |
| O1: Checkout and stock reliability | Duplicate-order protection, safe cancellations and clear price handling | **8,000–12,000** |
| O2: Collection register | Separate placed orders from recorded, checked payments | **4,000–8,000** |
| P1+P2: Automatic M-Pesa | One provider, verified confirmation, exception checks and controlled release | **21,000–35,000**, after O1/O2 |
| B1: One operations improvement | One selected email, delivery-rule or stock-history improvement | **3,000–6,000** |

**Suggested first approval: C1 at KES 3,000–5,000**, or **C1+C2 at KES 5,000–8,000** if affordable. This is a first correction, not a complete security clearance. C3/C4 remain important; C1–C4 together are KES 12,000–22,000, with safe temporary restrictions needed on unresolved risks.

Marketing plus the small homepage refresh is **KES 7,000–12,000**. Checkout reliability plus a manual collection register is **KES 12,000–20,000**. Automatic M-Pesa cannot responsibly be included in that same amount: its build/testing is additional and requires the earlier order controls. Keep the existing verified manual/COD arrangement until the full narrow payment path is ready.

There is no recommendation to buy every row now. Prices exclude hosting/provider charges, advertising, inventory, business staffing and taxes where applicable. Unexpected schema or historical-stock problems require a revised scope before extra spending. Full inclusions, exclusions, effort estimates and acceptance checks are in the supporting budget.

## 7. Current-scale monthly expenses
At five reported orders per month, there is no evidence that new expensive infrastructure is needed. Keep existing services where they are secure, durable and reliable. Hosting files describe Netlify/Render and an alternative server arrangement, but do not establish which bills the shop currently pays.

| Expense | Current-scale approach |
|---|---|
| Hosting, database, Redis and media | Confirm actual invoices first; retain suitable existing plans |
| Domain/DNS | Confirm annual renewal; divide by 12 for monthly planning |
| Essential backups and monitoring | Use included capabilities where adequate; **KES 0–1,000/month** extra allowance if needed |
| Transactional email | **KES 0–300/month** extra allowance, depending on existing service |
| Payment fees | Apply the actual agreed fee only to the relevant collected transactions |
| SMS | Optional; no new SMS subscription assumed |
| Technical maintenance | No mandatory retainer; optional **KES 1,000–3,000/month** only for agreed limited support |
| Google Ads | Keep separate from hosting; confirm current spend before deciding changes |
| Physical shop | Stock, rent, staff, transport and other expenses are separate and currently unknown |

These are allowances, not supplier quotations. Incremental backup/email overhead may be **KES 0–1,300/month** if the current setup is otherwise suitable; this is **not the total hosting bill** or a guarantee that free hosting is safe. If current services cannot protect/recover data, obtain the actual upgrade cost. Confirm existing expenditure before calculating a new monthly total.

## 8. How to fund improvements gradually
Set an improvement allowance only after stock replenishment and necessary bills are covered. The examples below are hypothetical; they do not state Zentora's actual profit.

| Hypothetical contribution after product/direct selling costs | Amount from KES 50,000 sales | Illustrative 20% improvement allocation |
|---|---:|---:|
| 10% | 5,000 | 1,000/month |
| 20% | 10,000 | 2,000/month |
| 30% | 15,000 | 3,000/month |

Rent, wages, tax and other shared expenses may still need to come out of that contribution. At the example saving rates, a KES 5,000 package takes roughly five, three or two months to fund. Urgent exposure must be restricted immediately while funds are arranged. Instalments can be agreed separately; they do not reduce the total price.

After each improvement, review visits, checkout starts, submitted orders, collected payments, cancellations, Ads spend and recurring manual work. Count unpaid COD orders separately from collected sales and prevent duplicate conversion reporting. With five monthly orders, one order changes the percentages considerably; no feature is guaranteed to increase sales or pay for itself.

## 9. Long-term capabilities
The full vision remains a reliable online retail platform: online payments, delivery/tracking, returns/refunds, customer notifications/support, usable administration, business reporting, campaign tools, better discovery and secure operation. Coupons, loyalty, marketing automation, additional integrations and more advanced analytics should follow demonstrated demand.

Independent sellers, commissions, seller dashboards and payouts are a separate future business expansion. They are not included in the initial proposal. The complete feature inventory and technical architecture are retained for future planning.

## 10. Owner's next decisions
1. Approve C1 first, and C2 if affordable; agree immediate restrictions for any unresolved critical issues.
2. Confirm hosting/domain invoices, backup access, actual delivery/payment procedures and a small affordable improvement budget.
3. Check the existing Ads conversion action and agree what counts as an order versus a collected sale.
4. Choose the next package only after the previous result is demonstrated and the remaining risks are safely handled.
5. Review progress and cash availability monthly before expanding scope.

This revision is based on the earlier code audit and newly supplied business estimates. It is not a fresh live-system certification. The prior discovery tests and frontend type check passed, but they did not establish complete payment, checkout or production security assurance. No application code or production system was changed.

Supporting detail: [budget and package definitions](ZENTORA_BUDGET_AND_COST_MODEL.md), [roadmap](ZENTORA_IMPLEMENTATION_ROADMAP.md), [complete feature inventory](ZENTORA_FEATURE_GAP_MATRIX.md), [technical findings](ZENTORA_TECHNICAL_AUDIT.md), [architecture](ZENTORA_ARCHITECTURE_PROPOSAL.md) and [evidence](ZENTORA_AUDIT_EVIDENCE.md).
