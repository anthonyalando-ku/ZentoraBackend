package orderusecase

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"testing"
	"zentora-service/internal/domain/delivery"
)

type deliveryStub struct{ fail bool }

func (d deliveryStub) Get(context.Context) (*delivery.Policy, error) {
	if d.fail {
		return nil, errors.New("unavailable")
	}
	p := &delivery.Policy{MethodID: 1, Version: 1, NairobiIndicativeFee: 300}
	p.Prepare()
	return p, nil
}
func TestDeliverySnapshotFallback(t *testing.T) {
	s := &Service{logger: zap.NewNop(), delivery: deliveryStub{fail: true}}
	got := s.deliverySnapshot(context.Background())
	if got.Notice != delivery.FallbackNotice || got.ConfirmedFee != nil {
		t.Fatal("policy failure did not fall back")
	}
	s.delivery = deliveryStub{}
	got = s.deliverySnapshot(context.Background())
	if got.PolicyVersion == nil || got.IncludedInOrderTotal {
		t.Fatal("policy not informational")
	}
}
