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
// adminEmail is the address that receives every new-order notification.
func NewOrderEmailSender(sender *EmailSender, adminEmail string) *OrderEmailSender {
	return &OrderEmailSender{
		sender:     sender,
		adminEmail: adminEmail,
	}
}

// SendOrderConfirmation sends the customer confirmation email.
// toEmail should be the email address captured at checkout or from the user profile.
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

// ─── HTML builders ────────────────────────────────────────────────────────────

func buildCustomerConfirmationBody(ord *order.Order) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
<h2>Thank you for your order!</h2>
<p>Hi %s,</p>
<p>We've received your order and it's now being processed. Here's a summary:</p>

<div class="box box-info">
  <strong>Order Number:</strong> %s<br/>
  <strong>Status:</strong> %s<br/>
  <strong>Currency:</strong> %s
</div>
`, ord.Shipping.FullName, ord.OrderNumber, strings.Title(strings.ToLower(string(ord.Status))), ord.Currency))

	// Items table
	sb.WriteString(`
<h2 style="font-size:17px;margin-top:28px;">Order Items</h2>
<table width="100%" cellpadding="10" cellspacing="0" style="border-collapse:collapse;font-size:14px;">
  <thead>
    <tr style="background-color:#f0f4ff;border-bottom:2px solid #e5e7eb;">
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
    <tr style="border-bottom:1px solid #e5e7eb;">
      <td>%s</td>
      <td align="center">%d</td>
      <td align="right">%s %.2f</td>
      <td align="right">%s %.2f</td>
    </tr>
`, item.ProductName, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}
	sb.WriteString(`  </tbody>
</table>
`)

	// Totals
	sb.WriteString(fmt.Sprintf(`
<table width="100%" cellpadding="6" cellspacing="0" style="font-size:14px;margin-top:16px;">
  <tr>
    <td align="right" style="color:#6b7280;">Subtotal</td>
    <td align="right" width="130"><strong>%s %.2f</strong></td>
  </tr>`, ord.Currency, ord.Subtotal))

	if ord.DiscountAmount > 0 {
		sb.WriteString(fmt.Sprintf(`
  <tr>
    <td align="right" style="color:#6b7280;">Discount</td>
    <td align="right"><strong style="color:#10b981;">- %s %.2f</strong></td>
  </tr>`, ord.Currency, ord.DiscountAmount))
	}

	sb.WriteString(fmt.Sprintf(`
  <tr style="border-top:2px solid #e5e7eb;">
    <td align="right"><strong style="font-size:15px;">Total</strong></td>
    <td align="right"><strong style="font-size:16px;color:#004b8f;">%s %.2f</strong></td>
  </tr>
</table>
`, ord.Currency, ord.TotalAmount))

	// Shipping
	sb.WriteString(fmt.Sprintf(`
<hr class="divider"/>
<h2 style="font-size:17px;">Shipping Address</h2>
<div class="box box-info">
  %s<br/>
  %s<br/>
`, ord.Shipping.FullName, ord.Shipping.AddressLine1))

	if ord.Shipping.AddressLine2 != nil && *ord.Shipping.AddressLine2 != "" {
		sb.WriteString(fmt.Sprintf("  %s<br/>\n", ord.Shipping.AddressLine2))
	}
	sb.WriteString(fmt.Sprintf(`  %s%s, %s<br/>
  %s<br/>
  Phone: %s
</div>
`,
		cityLine(ord.Shipping.Area), ord.Shipping.City, ord.Shipping.County,
		ord.Shipping.Country,
		ord.Shipping.Phone,
	))

	sb.WriteString(`
<p style="margin-top:24px;">We'll send you another email once your order has been shipped. If you have any questions, don't hesitate to reach out.</p>
<p>Thank you for shopping with <strong>Zentora</strong>!</p>
`)

	return sb.String()
}

func buildAdminNotificationBody(ord *order.Order) string {
	var sb strings.Builder

	userLabel := "Guest"
	if ord.UserID != nil {
		userLabel = fmt.Sprintf("User ID %d", *ord.UserID)
	}

	sb.WriteString(fmt.Sprintf(`
<h2>New Order Received</h2>
<div class="box box-warning">
  <strong>Order Number:</strong> %s<br/>
  <strong>Placed By:</strong> %s<br/>
  <strong>Status:</strong> %s<br/>
  <strong>Total Amount:</strong> %s %.2f
</div>
`, ord.OrderNumber, userLabel, strings.Title(strings.ToLower(string(ord.Status))), ord.Currency, ord.TotalAmount))

	// Items
	sb.WriteString(`
<h2 style="font-size:17px;margin-top:24px;">Items</h2>
<table width="100%" cellpadding="10" cellspacing="0" style="border-collapse:collapse;font-size:14px;">
  <thead>
    <tr style="background-color:#f0f4ff;border-bottom:2px solid #e5e7eb;">
      <th align="left">Product</th>
      <th align="left">Variant ID</th>
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
    <tr style="border-bottom:1px solid #e5e7eb;">
      <td>%s</td>
      <td>%s</td>
      <td align="center">%d</td>
      <td align="right">%s %.2f</td>
      <td align="right">%s %.2f</td>
    </tr>
`, item.ProductName, variantID, item.Quantity, ord.Currency, item.UnitPrice, ord.Currency, item.TotalPrice))
	}
	sb.WriteString(`  </tbody>
</table>
`)

	// Shipping
	sb.WriteString(fmt.Sprintf(`
<hr class="divider"/>
<h2 style="font-size:17px;">Shipping Details</h2>
<div class="credentials">
  <p><strong>Name:</strong> %s</p>
  <p><strong>Phone:</strong> %s</p>
  <p><strong>Address:</strong> %s`, ord.Shipping.FullName, ord.Shipping.Phone, ord.Shipping.AddressLine1))

	if ord.Shipping.AddressLine2 != nil && *ord.Shipping.AddressLine2 != "" {
		sb.WriteString(", " + *ord.Shipping.AddressLine2)
	}
	sb.WriteString(fmt.Sprintf(`</p>
  <p><strong>City/Area:</strong> %s%s</p>
  <p><strong>County:</strong> %s</p>
  <p><strong>Country:</strong> %s</p>
`,
		cityLine(ord.Shipping.Area), ord.Shipping.City,
		ord.Shipping.County,
		ord.Shipping.Country,
	))

	if ord.Shipping.PostalCode != nil && *ord.Shipping.PostalCode != "" {
		sb.WriteString(fmt.Sprintf("  <p><strong>Postal Code:</strong> %s</p>\n", ord.Shipping.PostalCode))
	}
	sb.WriteString("</div>\n")

	if ord.CartID != nil {
		sb.WriteString(fmt.Sprintf(`<p style="font-size:13px;color:#6b7280;">Originated from Cart ID: %d</p>`, *ord.CartID))
	}

	return sb.String()
}

// cityLine returns "Area, " when area is non-empty, otherwise "".
func cityLine(area *string) string {
	if area == nil || *area == "" {
		return ""
	}
	return *area + ", "
}