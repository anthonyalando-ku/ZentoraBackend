package email

import (
	"strings"
	"testing"
	"zentora-service/internal/domain/delivery"
	"zentora-service/internal/domain/order"
)

func TestDeliveryNoticeInTotals(t *testing.T) {
	o := &order.Order{Currency: "KES", Subtotal: 4500, TotalAmount: 4500, DeliveryInformation: delivery.NewSnapshot(nil)}
	o.DeliveryInformation.Notice = "Delivery <script>alert(1)</script>"
	text := buildTotals(o)
	if !strings.Contains(text, "excluding delivery") || !strings.Contains(text, "KES 4500.00") || strings.Contains(text, "<script>") {
		t.Fatal("incorrect or unescaped delivery totals", text)
	}
	o.DeliveryInformation = nil
	if strings.Contains(buildTotals(o), "excluding delivery") {
		t.Fatal("legacy order was relabelled")
	}
}
