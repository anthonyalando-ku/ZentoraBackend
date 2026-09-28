# Zentora: features to add and improve

This list describes shopping and shop-management features in everyday language. It is based on the existing review. “To add” means a complete working feature was not found in the website; the shop may already handle the task manually. Not everything needs to be added at once.

## Features to add

### Payments

- **M-Pesa payments:** customers receive a payment request on their phone and enter their PIN to pay.
- **Automatic payment confirmation:** the shop and customer can see when payment has actually been received.
- **Card payments:** customers can pay using a debit or credit card.
- **Payment progress and retry:** customers can see whether a payment is waiting, successful or unsuccessful, and try again when appropriate.
- **Payment matching:** match M-Pesa receipts and other payments to the correct orders.
- **Refund management:** record and follow full or partial refunds when an order is cancelled or a product is returned.

### Automatic marketing and promotions

- **Unfinished-cart reminders:** remind customers who added products but left without ordering, where they have agreed to receive messages.
- **Email and SMS offers:** send new arrivals, special offers and seasonal promotions to subscribed customers.
- **Newsletter signup:** allow visitors to subscribe to shop updates and unsubscribe easily.
- **Follow-up marketing:** send suitable offers to past customers or customers interested in particular products.
- **Discount codes:** customers enter a voucher code at checkout, with rules such as minimum spend and one use per customer.
- **Free-delivery offers:** apply free delivery when an order meets the advertised conditions.
- **Product bundles:** sell related products together at a combined price.
- **Loyalty rewards:** let repeat customers earn points or rewards.
- **Referral rewards:** reward customers for introducing new buyers.
- **Gift vouchers:** allow customers to buy a voucher for someone else.
- **Scheduled campaigns:** prepare an offer in advance and have it start and end on the chosen dates.
- **Offer pages:** create a dedicated page for a sale, seasonal campaign or selected product range.

### Website visits and advertising results

- **Google Analytics:** show how many people visit, which products they view and where they leave before ordering.
- **Website conversion tracking:** connect customer actions on the website to the existing Google Ads conversion setup.
- **Advertising results:** see which campaigns lead to orders and which orders become actual paid sales.
- **Shopping progress reports:** see how many visitors view products, add to cart, begin checkout and place an order.
- **Marketing preferences:** let customers choose whether to receive marketing messages and allow optional tracking.

Google Ads conversion tracking is reportedly already set up in the Ads account. What is missing from the reviewed website code is the connection that explicitly reports the relevant conversion event. That does not mean the Ads account necessarily records no conversions.

### Delivery and collection

- **Delivery charges at checkout:** show the applicable delivery fee before the customer commits to the order.
- **Delivery areas and rates:** let the owner set different charges for Nairobi and other supported destinations.
- **Delivery estimates:** tell customers when they can expect their order.
- **Shop pickup:** let customers choose to collect an order from the physical shop.
- **Shipment tracking:** show the progress of an order after it leaves the shop.
- **Courier connection:** send delivery details to a supported courier and receive tracking updates.
- **Delivery confirmation:** record that the customer received the goods.
- **Missed-delivery handling:** record unsuccessful deliveries and arrange another attempt.

### Customer service and order updates

- **Order-update messages:** notify customers when an order is confirmed, paid, dispatched, delivered or refunded.
- **Customer cancellation requests:** let customers request cancellation before an order is dispatched.
- **Return and exchange requests:** let customers select an order and explain which item they want to return or exchange.
- **Support enquiries linked to orders:** keep a customer's question, the reply and its resolution together.
- **Product questions and answers:** let customers ask questions before buying.
- **Receipts and invoices:** provide customer-ready documents showing the agreed charges and payment position.
- **Customer account controls:** allow customers to request a copy of their information or closure of their account.

### Stock and supplier management

Basic stock quantities and stock locations already exist. The additions are:

- **Low-stock alerts:** warn the owner when a product is running low.
- **Back-in-stock alerts:** notify interested customers when a sold-out product becomes available again.
- **Stock movement history:** show when stock was added, sold, returned or adjusted, and why.
- **Supplier records:** keep supplier contacts and the products they supply.
- **Supplier purchase orders:** record what the shop has ordered and what has arrived.
- **Stock-count checks:** compare stock recorded on the website with the quantities counted in the shop.
- **Returned-stock handling:** record whether a returned item can be sold again.
- **Stock performance reports:** identify fast-selling products, slow-moving stock and items that need replenishing.

### Product and shop management

- **Bulk product uploads:** add many products from a spreadsheet.
- **Bulk updates:** change prices or product details for several items together.
- **Product exports:** download product information for checking or record-keeping.
- **Product comparison:** let customers compare similar items side by side.
- **Product videos:** include demonstrations or short videos alongside product photos.
- **Scheduled product launches:** make new products visible on a chosen date.
- **Banner editing by the owner:** change homepage images, offer wording and links without requesting a website code change.
- **Buying guides:** publish simple advice that helps customers choose the right product.
- **Order notes and activity history:** record customer instructions, staff actions and reasons for changes.
- **Part-order delivery:** record when some items have been delivered and others will follow.

### Business reports

- **Collected-sales reports:** show money actually received separately from orders merely placed.
- **Payment and refund reports:** show outstanding payments, money collected and refunds issued.
- **Product sales reports:** show which products generate orders and collected sales.
- **Customer reports:** show new customers, repeat customers and average order value.
- **Delivery reports:** show completed, delayed and unsuccessful deliveries.
- **Profit reports:** compare sales with product costs, payment fees and delivery costs once those costs are recorded.
- **Downloadable reports:** export selected sales, stock and payment records for checking.

## Existing features that need fixing or completing

| Existing feature | What needs improvement |
|---|---|
| **Inventory management** | Cancelled orders should return held stock to available stock. Stock should also update correctly as orders are fulfilled. |
| **Google Merchant Center feed** | Correct the stock calculation so available products are not wrongly shown as unavailable on Google Shopping. |
| **Product prices and discounts** | Ensure the price advertised, the discount shown and the amount recorded at checkout agree. |
| **Checkout totals** | Make delivery charges and any applicable taxes clear; customers should not receive unexpected additional charges. |
| **Order placement** | Prevent a repeated click or a retry from creating the same order more than once. |
| **Order processing** | Make changes such as cancellation, dispatch and delivery follow the correct order and update stock appropriately. |
| **Cash on delivery** | Record whether money was actually collected instead of relying only on the order's status. |
| **Customer confirmation emails** | Complete the existing email feature so customers receive their order confirmation reliably. |
| **Customer order history** | Ensure each customer can see only their own orders and delivery details. |
| **Staff access** | Ensure staff can only perform the actions the owner has allowed. |
| **Sales dashboard** | Separate submitted orders, collected payments, cancellations and refunds so the figures are not misleading. |
| **Google Ads measurement** | Connect and check the website's conversion events; avoid counting one order twice or treating an unpaid order as a paid sale. |
| **Homepage promotions** | Make offers easier to update and ensure banners describe discounts and delivery promises the shop can actually honour. |

### Existing features to check before expanding them

- **Saved shopping baskets:** confirm that customers do not lose their selected items when they sign in.
- **Product reviews:** check that customers can complete the review process and that reviews are linked to genuine purchases.
- **Search and recommendations:** check whether results help customers find relevant, available products.
- **Website backups:** confirm that shop information and product images can be restored if something goes wrong.

These checks are not claims that each feature is broken.

## Optional future expansion

- **Third-party sellers:** allow other businesses to sell through Zentora.
- **Seller dashboards:** let sellers manage their products, stock and orders.
- **Seller commissions and payments:** calculate Zentora's share and the amount owed to each seller.
- **Seller reviews and disputes:** manage seller performance and resolve problems between buyers and sellers.

These are a future expansion, not a requirement for running Zentora's current shop.

