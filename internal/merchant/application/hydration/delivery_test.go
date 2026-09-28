package hydration

import (
	"testing"
	merchant "zentora-service/internal/merchant/domain"
)

func TestInformationalDeliveryOmitsFixedRate(t *testing.T) {
	cfg := merchant.HydrationConfig{DefaultCountry: "KE", DefaultCurrency: "KES", OmitShippingRate: true, DefaultShippingFee: merchant.MoneyFromDecimal(200, "KES")}
	if len(buildShipping(cfg)) != 0 {
		t.Fatal("informational delivery advertised a fixed fee")
	}
}
