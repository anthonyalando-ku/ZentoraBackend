package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"zentora-service/internal/domain/delivery"
)

type DeliveryRepository struct{ db *pgxpool.Pool }

func NewDeliveryRepository(db *pgxpool.Pool) *DeliveryRepository { return &DeliveryRepository{db: db} }

const deliveryPolicyQuery = `SELECT m.id, m.policy_version, m.pricing_mode, r.currency,
 r.base_fee::bigint, COALESCE(m.description, ''), m.updated_at
 FROM shipping_methods m
 JOIN shipping_method_rates r ON r.shipping_method_id = m.id
 JOIN shipping_zones z ON z.id = r.zone_id
 WHERE m.code = 'standard-delivery' AND m.is_active AND m.pricing_mode = 'informational'
 AND z.code = 'ke-nairobi' AND z.region_code = 'KE-30' AND z.is_active
 AND r.rate_type = 'indicative' AND r.currency = 'KES' AND r.base_fee > 0
 AND EXISTS (SELECT 1 FROM shipping_zone_countries c WHERE c.zone_id = z.id AND c.country = 'KE')
 AND EXISTS (SELECT 1 FROM shipping_method_rates nr JOIN shipping_zones nz ON nz.id = nr.zone_id
 JOIN shipping_zone_countries nc ON nc.zone_id = nz.id AND nc.country = 'KE'
 WHERE nr.shipping_method_id = m.id AND nz.code = 'ke-nationwide' AND nz.is_active
 AND nr.rate_type = 'quote_required' AND nr.base_fee IS NULL AND nr.currency = 'KES')`

func (r *DeliveryRepository) Get(ctx context.Context) (*delivery.Policy, error) {
	var p delivery.Policy
	err := r.db.QueryRow(ctx, deliveryPolicyQuery).Scan(&p.MethodID, &p.Version, &p.PricingMode, &p.Currency, &p.NairobiIndicativeFee, &p.AdditionalInformation, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Prepare()
	return &p, nil
}

func (r *DeliveryRepository) Update(ctx context.Context, u delivery.Update, adminID int64) (*delivery.Policy, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `UPDATE shipping_methods SET description=$1, policy_version=policy_version+1,
 updated_at=NOW(), updated_by=$2 WHERE code='standard-delivery' AND pricing_mode='informational'
 AND is_active AND policy_version=$3 RETURNING id`, strings.TrimSpace(u.AdditionalInformation), adminID, u.ExpectedVersion).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, delivery.ErrConflict
		}
		return nil, err
	}
	result, err := tx.Exec(ctx, `UPDATE shipping_method_rates SET base_fee=$1
 WHERE shipping_method_id=$2 AND zone_id=(SELECT id FROM shipping_zones WHERE code='ke-nairobi' AND is_active)
 AND rate_type='indicative' AND currency='KES'`, u.NairobiIndicativeFee, id)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() != 1 {
		return nil, delivery.ErrInvalid
	}
	var p delivery.Policy
	if err := tx.QueryRow(ctx, deliveryPolicyQuery).Scan(&p.MethodID, &p.Version, &p.PricingMode, &p.Currency, &p.NairobiIndicativeFee, &p.AdditionalInformation, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	p.Prepare()
	return &p, nil
}
