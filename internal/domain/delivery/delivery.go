package delivery

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const FallbackNotice = "Delivery charges are confirmed separately and are not included in the order total."

var ErrConflict = errors.New("delivery settings changed; reload before saving")
var ErrInvalid = errors.New("invalid delivery settings")

type Policy struct {
	MethodID              int64     `json:"method_id"`
	Version               int       `json:"version"`
	PricingMode           string    `json:"pricing_mode"`
	Currency              string    `json:"currency"`
	NairobiIndicativeFee  int64     `json:"nairobi_indicative_fee"`
	AdditionalInformation string    `json:"additional_information"`
	Message               string    `json:"message"`
	Coverage              string    `json:"coverage"`
	IncludedInOrderTotal  bool      `json:"included_in_order_total"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Update struct {
	ExpectedVersion       int    `json:"expected_version"`
	NairobiIndicativeFee  int64  `json:"nairobi_indicative_fee"`
	AdditionalInformation string `json:"additional_information"`
}

func (u Update) Validate() error {
	if u.ExpectedVersion < 1 || u.NairobiIndicativeFee < 1 || u.NairobiIndicativeFee > 100000 || utf8.RuneCountInString(u.AdditionalInformation) > 1000 {
		return ErrInvalid
	}
	return nil
}

func (p *Policy) Prepare() {
	p.Coverage = "Kenya nationwide"
	p.IncludedInOrderTotal = false
	p.Message = fmt.Sprintf("We deliver across Kenya. Standard Nairobi delivery is typically KES %d; the actual fee depends on your location. We will contact you to confirm delivery charges. Delivery is not included in your order total.", p.NairobiIndicativeFee)
	if note := strings.TrimSpace(p.AdditionalInformation); note != "" {
		p.Message += " " + note
	}
}

type Snapshot struct {
	PolicyVersion        *int   `json:"policy_version"`
	MethodID             *int64 `json:"method_id"`
	PricingStatus        string `json:"pricing_status"`
	IncludedInOrderTotal bool   `json:"included_in_order_total"`
	ConfirmedFee         *int64 `json:"confirmed_fee"`
	Currency             string `json:"currency"`
	Notice               string `json:"notice"`
}

func NewSnapshot(p *Policy) *Snapshot {
	s := &Snapshot{PricingStatus: "pending_confirmation", Currency: "KES", Notice: FallbackNotice}
	if p != nil {
		version, methodID := p.Version, p.MethodID
		s.PolicyVersion, s.MethodID, s.Notice = &version, &methodID, p.Message
	}
	return s
}
