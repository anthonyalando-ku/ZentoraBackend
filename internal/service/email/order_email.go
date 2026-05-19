package email

import (
	"fmt"
	"strings"

	"zentora-service/internal/domain/order"
)

// OrderEmailSender sends order-related emails to customers and admins.
type OrderEmailSender struct {
	sender     *EmailSender
	adminEmail string
}

// NewOrderEmailSender creates an OrderEmailSender.
func NewOrderEmailSender(sender *EmailSender, adminEmail string) *OrderEmailSender {
	return &OrderEmailSender{sender: sender, adminEmail: adminEmail}
}

// SendOrderConfirmation sends the customer confirmation email.
func (o *OrderEmailSender) SendOrderConfirmation(toEmail string, ord *order.Order) error {
	if toEmail == "" || ord == nil {
		return fmt.Errorf("order email: missing recipient or order")
	}
	subject := fmt.Sprintf("Order Confirmed – %s", ord.OrderNumber)
	body := buildCustomerConfirmationBody(ord)
	if err := o.sender.Send(toEmail, subject, body); err != nil {
		return fmt.Errorf("order confirmation to customer: %w", err)
	}
	return nil
}

// SendAdminOrderNotification sends a new-order alert to the configured admin address.
func (o *OrderEmailSender) SendAdminOrderNotification(ord *order.Order) error {
	if ord == nil {
		return fmt.Errorf("order email: nil order")
	}
	subject := fmt.Sprintf("🛒 New Order – %s (KES %.2f)", ord.OrderNumber, ord.TotalAmount)
	body := buildAdminNotificationBody(ord)
	if err := o.sender.Send(o.adminEmail, subject, body); err != nil {
		return fmt.Errorf("order notification to admin: %w", err)
	}
	return nil
}

// ─── shared CSS injected once per email ──────────────────────────────────────

const emailStyles = `
<style>
  /* Reset */
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
         font-size: 14px; color: #1f2937; background: #f3f4f6; }

  .wrapper   { max-width: 620px; margin: 0 auto; background: #fff;
               border-radius: 12px; overflow: hidden; }
  .header    { background: #004b8f; padding: 28px 24px; text-align: center; }
  .header h1 { color: #fff; font-size: 22px; font-weight: 700; letter-spacing: -.3px; }
  .header p  { color: rgba(255,255,255,.75); font-size: 13px; margin-top: 4px; }
  .body      { padding: 28px 24px; }

  /* Info box */
  .meta-box  { background: #f0f4ff; border: 1px solid #dbe4ff; border-radius: 10px;
               padding: 16px 18px; margin-bottom: 24px; }
  .meta-box .row { display: flex; justify-content: space-between; align-items: flex-start;
                    padding: 5px 0; gap: 8px; }
  .meta-box .label { color: #6b7280; font-size: 12px; white-space: nowrap; }
  .meta-box .value { font-weight: 600; font-size: 13px; color: #111827;
                     text-align: right; word-break: break-all; }

  /* Section heading */
  .section-title { font-size: 15px; font-weight: 700; color: #111827;
                   margin: 0 0 12px; padding-bottom: 8px;
                   border-bottom: 2px solid #e5e7eb; }

  /* ── Desktop table (hidden on mobile) ── */
  .items-table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .items-table thead tr { background: #f0f4ff; }
  .items-table th { padding: 10px 12px; font-weight: 600; color: #374151;
                    border-bottom: 2px solid #dbe4ff; }
  .items-table td { padding: 10px 12px; border-bottom: 1px solid #f3f4f6;
                    vertical-align: top; }
  .items-table td.name { font-weight: 500; }
  .items-table tr:last-child td { border-bottom: none; }

  /* ── Mobile item card (hidden on desktop) ── */
  .item-card { display: none; }

  /* ── Totals ── */
  .totals { width: 100%; font-size: 13px; margin-top: 4px; }
  .totals td { padding: 5px 0; }
  .totals .lbl { color: #6b7280; }
  .totals .amt { text-align: right; font-weight: 600; }
  .totals .grand td { border-top: 2px solid #e5e7eb; padding-top: 10px; }
  .totals .grand .lbl { font-size: 15px; font-weight: 700; color: #111827; }
  .totals .grand .amt { font-size: 16px; color: #004b8f; }

  /* Shipping */
  .shipping-grid { font-size: 13px; line-height: 1.7; }
  .shipping-grid .row { display: flex; gap: 8px; padding: 3px 0; }
  .shipping-grid .lbl { color: #6b7280; min-width: 90px; flex-shrink: 0; }
  .shipping-grid .val { font-weight: 500; word-break: break-word; }

  /* Footer */
  .footer { background: #f9fafb; border-top: 1px solid #e5e7eb;
            padding: 20px 24px; font-size: 12px; color: #9ca3af; text-align: center; }
  .footer a { color: #004b8f; text-decoration: none; }
  .divider { border: none; border-top: 1px solid #e5e7eb; margin: 24px 0; }

  /* ── Responsive ── */
  @media (max-width: 600px) {
    .body { padding: 20px 16px; }
    .items-table { display: none; }
    .item-card { display: block; border: 1px solid #e5e7eb; border-radius: 10px;
                 padding: 14px; margin-bottom: 10px; }
    .item-card .name { font-weight: 600; font-size: 14px; margin-bottom: 10px;
                       color: #111827; line-height: 1.4; }
    .item-card .meta { display: flex; justify-content: space-between;
                       font-size: 12px; gap: 6px; flex-wrap: wrap; }
    .item-card .meta-item { background: #f3f4f6; border-radius: 6px;
                             padding: 5px 10px; text-align: center; flex: 1; }
    .item-card .meta-item .k { color: #6b7280; display: block; margin-bottom: 2px; }
    .item-card .meta-item .v { font-weight: 700; color: #111827; }
    .item-card .meta-item.total .v { color: #004b8f; }
    .meta-box .row { flex-direction: column; gap: 1px; }
    .meta-box .value { text-align: left; }
    .shipping-grid .lbl { min-width: 75px; }
  }
</style>
`

// ─── HTML builders ────────────────────────────────────────────────────────────

func buildCustomerConfirmationBody(ord *order.Order) string {
	var sb strings.Builder

	sb.WriteString(emailStyles)
	sb.WriteString(`<div class="wrapper">`)

	// Header
	sb.WriteString(fmt.Sprintf(`
<div class="header">
  <h1>Order Confirmed ✓</h1>
  <p>Hi %s, we've received your order and it's being processed.</p>
</div>
<div class="body">
`, ord.Shipping.FullName))

	// Meta box
	sb.WriteString(fmt.Sprintf(`
<div class="meta-box">
  <div class="row"><span class="label">Order Number</span><span class="value">%s</span></div>
  <div class="row"><span class="label">Status</span><span class="value">%s</span></div>
  <div class="row"><span class="label">Currency</span><span class="value">%s</span></div>
</div>
`, ord.OrderNumber, titleCase(string(ord.Status)), ord.Currency))

	// Items section
	sb.WriteString(`<p class="section-title">Order Items</p>`)
	sb.WriteString(buildItemsTableAndCards(ord))

	// Totals
	sb.WriteString(buildTotals(ord))

	// Shipping
	sb.WriteString(`<hr class="divider"/>`)
	sb.WriteString(`<p class="section-title">Shipping Address</p>`)
	sb.WriteString(buildShipping(ord))

	sb.WriteString(`
<p style="margin-top:24px;font-size:13px;line-height:1.6;color:#4b5563;">
  We'll send you another email once your order ships.
  Questions? Just reply to this email or reach out to our support team.
</p>
<p style="margin-top:16px;font-size:13px;color:#4b5563;">
  Thank you for shopping with <strong style="color:#004b8f;">Zentora</strong>!
</p>
</div>`) // close .body

	sb.WriteString(`
<div class="footer">
  &copy; Zentora. All rights reserved.<br/>
  <a href="#">Unsubscribe</a> &nbsp;·&nbsp; <a href="#">Privacy Policy</a>
</div>
</div>`) // close .wrapper

	return sb.String()
}

func buildAdminNotificationBody(ord *order.Order) string {
	var sb strings.Builder

	userLabel := "Guest"
	if ord.UserID != nil {
		userLabel = fmt.Sprintf("User ID %d", *ord.UserID)
	}

	sb.WriteString(emailStyles)
	sb.WriteString(`<div class="wrapper">`)

	// Header
	sb.WriteString(`
<div class="header" style="background:#7c3aed;">
  <h1>🛒 New Order</h1>
  <p>A new order has been placed and needs attention.</p>
</div>
<div class="body">
`)

	// Meta box (warning colour)
	sb.WriteString(fmt.Sprintf(`
<div class="meta-box" style="background:#fffbeb;border-color:#fde68a;">
  <div class="row"><span class="label">Order Number</span><span class="value">%s</span></div>
  <div class="row"><span class="label">Placed By</span><span class="value">%s</span></div>
  <div class="row"><span class="label">Status</span><span class="value">%s</span></div>
  <div class="row"><span class="label">Total</span>
    <span class="value" style="color:#004b8f;">%s %.2f</span></div>
</div>
`, ord.OrderNumber, userLabel, titleCase(string(ord.Status)), ord.Currency, ord.TotalAmount))

	// Items — admin gets variant IDs too
	sb.WriteString(`<p class="section-title">Items</p>`)
	sb.WriteString(buildAdminItemsTableAndCards(ord))

	// Shipping
	sb.WriteString(`<hr class="divider"/>`)
	sb.WriteString(`<p class="section-title">Shipping Details</p>`)
	sb.WriteString(buildShipping(ord))

	if ord.CartID != nil {
		sb.WriteString(fmt.Sprintf(
			`<p style="font-size:12px;color:#9ca3af;margin-top:16px;">Originated from Cart ID: %d</p>`,
			*ord.CartID,
		))
	}

	sb.WriteString(`</div>`) // close .body
	sb.WriteString(`
<div class="footer">
  Zentora Admin Notification &nbsp;·&nbsp; Do not share this email.
</div>
</div>`) // close .wrapper

	return sb.String()
}

// ─── Shared sub-builders ──────────────────────────────────────────────────────

// buildItemsTableAndCards renders both the desktop table and mobile cards for customers.
func buildItemsTableAndCards(ord *order.Order) string {
	var sb strings.Builder

	// Desktop table
	sb.WriteString(`
<table class="items-table">
  <thead>
    <tr>
      <th align="left">Product</th>
      <th align="center">Qty</th>
      <th align="right">Unit Price</th>
      <th align="right">Total</th>
    </tr>
  </thead>
  <tbody>
`)
	for _, item := range ord.Items {
		sb.WriteString(fmt.Sprintf(`
    <tr>
      <td class="name">%s</td>
      <td align="center">%d</td>
      <td align="right">%s %.2f</td>
      <td align="right" style="font-weight:600;color:#004b8f;">%s %.2f</td>
    </tr>
`, item.ProductName, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}
	sb.WriteString(`  </tbody></table>`)

	// Mobile cards
	for _, item := range ord.Items {
		sb.WriteString(fmt.Sprintf(`
<div class="item-card">
  <div class="name">%s</div>
  <div class="meta">
    <div class="meta-item">
      <span class="k">Qty</span>
      <span class="v">%d</span>
    </div>
    <div class="meta-item">
      <span class="k">Unit</span>
      <span class="v">%s %.2f</span>
    </div>
    <div class="meta-item total">
      <span class="k">Total</span>
      <span class="v">%s %.2f</span>
    </div>
  </div>
</div>
`, item.ProductName, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}

	return sb.String()
}

// buildAdminItemsTableAndCards adds a Variant ID column for admin emails.
func buildAdminItemsTableAndCards(ord *order.Order) string {
	var sb strings.Builder

	sb.WriteString(`
<table class="items-table">
  <thead>
    <tr>
      <th align="left">Product</th>
      <th align="left">Variant</th>
      <th align="center">Qty</th>
      <th align="right">Unit Price</th>
      <th align="right">Line Total</th>
    </tr>
  </thead>
  <tbody>
`)
	for _, item := range ord.Items {
		variantID := "—"
		if item.VariantID != nil {
			variantID = fmt.Sprintf("%d", *item.VariantID)
		}
		sb.WriteString(fmt.Sprintf(`
    <tr>
      <td class="name">%s</td>
      <td style="color:#6b7280;">%s</td>
      <td align="center">%d</td>
      <td align="right">%s %.2f</td>
      <td align="right" style="font-weight:600;color:#004b8f;">%s %.2f</td>
    </tr>
`, item.ProductName, variantID, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}
	sb.WriteString(`  </tbody></table>`)

	// Mobile cards (include variant)
	for _, item := range ord.Items {
		variantID := "—"
		if item.VariantID != nil {
			variantID = fmt.Sprintf("%d", *item.VariantID)
		}
		sb.WriteString(fmt.Sprintf(`
<div class="item-card">
  <div class="name">%s</div>
  <div class="meta">
    <div class="meta-item">
      <span class="k">Variant</span>
      <span class="v">%s</span>
    </div>
    <div class="meta-item">
      <span class="k">Qty</span>
      <span class="v">%d</span>
    </div>
    <div class="meta-item">
      <span class="k">Unit</span>
      <span class="v">%s %.2f</span>
    </div>
    <div class="meta-item total">
      <span class="k">Total</span>
      <span class="v">%s %.2f</span>
    </div>
  </div>
</div>
`, item.ProductName, variantID, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}

	return sb.String()
}

func buildTotals(ord *order.Order) string {
	var sb strings.Builder
	sb.WriteString(`<table class="totals" style="margin-top:16px;">`)
	sb.WriteString(fmt.Sprintf(`
  <tr>
    <td class="lbl">Subtotal</td>
    <td class="amt">%s %.2f</td>
  </tr>`, ord.Currency, ord.Subtotal))

	if ord.DiscountAmount > 0 {
		sb.WriteString(fmt.Sprintf(`
  <tr>
    <td class="lbl">Discount</td>
    <td class="amt" style="color:#10b981;">− %s %.2f</td>
  </tr>`, ord.Currency, ord.DiscountAmount))
	}

	sb.WriteString(fmt.Sprintf(`
  <tr class="grand">
    <td class="lbl">Total</td>
    <td class="amt">%s %.2f</td>
  </tr>
</table>`, ord.Currency, ord.TotalAmount))
	return sb.String()
}

func buildShipping(ord *order.Order) string {
	var sb strings.Builder
	sb.WriteString(`<div class="shipping-grid">`)

	rows := []struct{ k, v string }{
		{"Name", ord.Shipping.FullName},
		{"Phone", ord.Shipping.Phone},
		{"Address", ord.Shipping.AddressLine1},
	}
	shippingCounty := ord.Shipping.City
	if ord.Shipping.County != nil && *ord.Shipping.County != "" {
		shippingCounty = *ord.Shipping.County
	}
	if ord.Shipping.AddressLine2 != nil && *ord.Shipping.AddressLine2 != "" {
		rows = append(rows, struct{ k, v string }{"", *ord.Shipping.AddressLine2})
	}
	rows = append(rows,
		struct{ k, v string }{"City / Area", cityLine(ord.Shipping.Area) + ord.Shipping.City},
		struct{ k, v string }{"County", shippingCounty},
		struct{ k, v string }{"Country", ord.Shipping.Country},
	)
	if ord.Shipping.PostalCode != nil && *ord.Shipping.PostalCode != "" {
		rows = append(rows, struct{ k, v string }{"Postal Code", *ord.Shipping.PostalCode})
	}

	for _, r := range rows {
		sb.WriteString(fmt.Sprintf(
			`<div class="row"><span class="lbl">%s</span><span class="val">%s</span></div>`,
			r.k, r.v,
		))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func titleCase(s string) string {
	return strings.Title(strings.ToLower(s)) //nolint:staticcheck
}

// cityLine returns "Area, " when area is non-empty, otherwise "".
func cityLine(area *string) string {
	if area == nil || *area == "" {
		return ""
	}
	return *area + ", "
}