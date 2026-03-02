// internal/service/user/service.go
package user

import (
	"context"
	"fmt"

	"zentora-service/internal/domain/user"
	"zentora-service/internal/repository/postgres"
)

// UserService provides business logic for user-related operations.
type UserService struct {
	addressRepo *postgres.UserAddressRepository
}

// NewUserService creates a new UserService.
func NewUserService(addressRepo *postgres.UserAddressRepository) *UserService {
	return &UserService{addressRepo: addressRepo}
}

func (s *UserService) CreateAddress(ctx context.Context, userID int64, req *user.CreateAddressRequest) (*user.Address, error) {
	a := &user.Address{
		UserID:       userID,
		FullName:     req.FullName,
		PhoneNumber:  req.PhoneNumber,
		Country:      req.Country,
		County:       req.County,
		City:         req.City,
		Area:         req.Area,
		PostalCode:   req.PostalCode,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		IsDefault:    req.IsDefault,
	}
	if err := s.addressRepo.CreateAddress(ctx, a); err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}
	// If flagged as default, promote it
	if a.IsDefault {
		if err := s.addressRepo.SetDefaultAddress(ctx, userID, a.ID); err != nil {
			return nil, fmt.Errorf("set default address: %w", err)
		}
	}
	return a, nil
}

func (s *UserService) GetAddressByID(ctx context.Context, userID, id int64) (*user.Address, error) {
	a, err := s.addressRepo.GetAddressByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, fmt.Errorf("address not found")
	}
	return a, nil
}

func (s *UserService) ListAddresses(ctx context.Context, userID int64) ([]user.Address, error) {
	return s.addressRepo.ListAddressesByUser(ctx, userID)
}

func (s *UserService) UpdateAddress(ctx context.Context, userID, id int64, req *user.UpdateAddressRequest) (*user.Address, error) {
	a, err := s.addressRepo.GetAddressByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, fmt.Errorf("address not found")
	}

	if req.FullName != nil {
		a.FullName = *req.FullName
	}
	if req.PhoneNumber != nil {
		a.PhoneNumber = *req.PhoneNumber
	}
	if req.Country != nil {
		a.Country = *req.Country
	}
	if req.County != nil {
		a.County = req.County
	}
	if req.City != nil {
		a.City = *req.City
	}
	if req.Area != nil {
		a.Area = req.Area
	}
	if req.PostalCode != nil {
		a.PostalCode = req.PostalCode
	}
	if req.AddressLine1 != nil {
		a.AddressLine1 = *req.AddressLine1
	}
	if req.AddressLine2 != nil {
		a.AddressLine2 = req.AddressLine2
	}
	if req.IsDefault != nil {
		a.IsDefault = *req.IsDefault
	}

	if err := s.addressRepo.UpdateAddress(ctx, a); err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}
	// Promote if needed
	if a.IsDefault {
		if err := s.addressRepo.SetDefaultAddress(ctx, userID, a.ID); err != nil {
			return nil, fmt.Errorf("set default address: %w", err)
		}
	}
	return a, nil
}

func (s *UserService) DeleteAddress(ctx context.Context, userID, id int64) error {
	a, err := s.addressRepo.GetAddressByID(ctx, id)
	if err != nil {
		return err
	}
	if a.UserID != userID {
		return fmt.Errorf("address not found")
	}
	return s.addressRepo.DeleteAddress(ctx, id)
}

func (s *UserService) SetDefaultAddress(ctx context.Context, userID, id int64) error {
	a, err := s.addressRepo.GetAddressByID(ctx, id)
	if err != nil {
		return err
	}
	if a.UserID != userID {
		return fmt.Errorf("address not found")
	}
	return s.addressRepo.SetDefaultAddress(ctx, userID, id)
}
