// internal/domain/user/repository.go
package user

import "context"

// AddressRepository defines persistence operations for user addresses.
type AddressRepository interface {
	CreateAddress(ctx context.Context, a *Address) error
	GetAddressByID(ctx context.Context, id int64) (*Address, error)
	ListAddressesByUser(ctx context.Context, userID int64) ([]Address, error)
	UpdateAddress(ctx context.Context, a *Address) error
	DeleteAddress(ctx context.Context, id int64) error
	SetDefaultAddress(ctx context.Context, userID, addressID int64) error
}
