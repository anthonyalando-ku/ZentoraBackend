package delivery

import (
	"strings"
	"testing"
)

func TestInformationalSnapshot(t *testing.T) {
	p := &Policy{MethodID: 7, Version: 1, NairobiIndicativeFee: 300}
	p.Prepare()
	s := NewSnapshot(p)
	if s.IncludedInOrderTotal || s.ConfirmedFee != nil || !strings.Contains(s.Notice, "KES 300") {
		t.Fatalf("not an informational snapshot: %+v", s)
	}
	p.Version = 2
	p.NairobiIndicativeFee = 500
	p.Prepare()
	if *s.PolicyVersion != 1 || !strings.Contains(s.Notice, "KES 300") {
		t.Fatal("saved snapshot changed with policy")
	}
	fallback := NewSnapshot(nil)
	if fallback.PolicyVersion != nil || fallback.ConfirmedFee != nil || fallback.Notice != FallbackNotice {
		t.Fatal("fallback invented delivery pricing")
	}
}

func TestValidateSettings(t *testing.T) {
	for _, u := range []Update{{0, 300, ""}, {1, 0, ""}, {1, -1, ""}, {1, 100001, ""}, {1, 300, strings.Repeat("a", 1001)}} {
		if u.Validate() == nil {
			t.Fatalf("accepted invalid update: %+v", u)
		}
	}
	if (Update{1, 300, "Contact us for delivery details."}).Validate() != nil {
		t.Fatal("valid update rejected")
	}
}
