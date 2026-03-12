package cartusecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zentora-service/internal/domain/cart"
	cartrepo "zentora-service/internal/repository/cart"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo  cartrepo.Repository
	redis *redis.Client
}

func NewService(repo cartrepo.Repository, redis *redis.Client) *Service {
	return &Service{repo: repo, redis: redis}
}

func (s *Service) cacheKey(userID int64) string {
	return fmt.Sprintf("zentora:cart:active:user:%d", userID)
}

const cartTTL = 15 * time.Second

func (s *Service) invalidate(ctx context.Context, userID int64) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, s.cacheKey(userID)).Err()
}

func (s *Service) GetActiveCart(ctx context.Context, userID int64) (*cart.Cart, error) {
	if userID <= 0 {
		return nil, cart.ErrInvalidInput
	}

	// Cache-first (hot path: reads)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, s.cacheKey(userID)).Bytes(); err == nil && len(raw) > 0 {
			var c cart.Cart
			if err := json.Unmarshal(raw, &c); err == nil {
				return &c, nil
			}
		}
	}

	c, err := s.repo.GetActiveCartWithItemsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache (best-effort)
	if s.redis != nil && c != nil {
		if raw, err := json.Marshal(c); err == nil {
			_ = s.redis.Set(ctx, s.cacheKey(userID), raw, cartTTL).Err()
		}
	}

	return c, nil
}

// AddOrUpdateItem upserts a cart item (variant required) and returns the updated cart.
// NOTE: price_at_added should come from catalog pricing service in a later step.
// For now it must be provided by caller or defaulted by handler/usecase integration.
func (s *Service) AddOrUpdateItem(ctx context.Context, userID int64, in cart.UpsertCartItemInput) (*cart.Cart, error) {
	if userID <= 0 {
		return nil, cart.ErrInvalidInput
	}
	if in.VariantID <= 0 {
		return nil, cart.ErrVariantRequired
	}
	if in.ProductID <= 0 || in.Quantity <= 0 {
		return nil, cart.ErrInvalidInput
	}
	if in.PriceAtAdded == "" {
		return nil, fmt.Errorf("%w: price_at_added is required", cart.ErrInvalidInput)
	}

	c, err := s.repo.GetOrCreateActiveCartForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpsertCartItem(ctx, c.ID, in); err != nil {
		return nil, err
	}

	s.invalidate(ctx, userID)
	return s.GetActiveCart(ctx, userID)
}

func (s *Service) RemoveItem(ctx context.Context, userID int64, itemID int64) (*cart.Cart, error) {
	if userID <= 0 || itemID <= 0 {
		return nil, cart.ErrInvalidInput
	}

	c, err := s.repo.GetActiveCartWithItemsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, cart.ErrCartNotFound
	}

	if err := s.repo.RemoveCartItem(ctx, c.ID, itemID); err != nil {
		return nil, cart.ErrCartItemNotFound
	}

	s.invalidate(ctx, userID)
	return s.GetActiveCart(ctx, userID)
}

func (s *Service) ClearActiveCart(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return cart.ErrInvalidInput
	}

	c, err := s.repo.GetActiveCartWithItemsForUser(ctx, userID)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}

	if err := s.repo.ClearCart(ctx, c.ID); err != nil {
		return err
	}

	s.invalidate(ctx, userID)
	return nil
}