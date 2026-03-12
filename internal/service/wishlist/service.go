package wishlistusecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zentora-service/internal/domain/wishlist"
	wishlistrepo "zentora-service/internal/repository/wishlist"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo  wishlistrepo.Repository
	redis *redis.Client
}

func NewService(repo wishlistrepo.Repository, redis *redis.Client) *Service {
	return &Service{repo: repo, redis: redis}
}

func (s *Service) key(userID int64) string {
	return fmt.Sprintf("zentora:wishlist:user:%d", userID)
}

const ttl = 30 * time.Second

func (s *Service) invalidate(ctx context.Context, userID int64) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, s.key(userID)).Err()
}

func (s *Service) GetMyWishlist(ctx context.Context, userID int64) (*wishlist.Wishlist, error) {
	if userID <= 0 {
		return nil, wishlist.ErrInvalidInput
	}

	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, s.key(userID)).Bytes(); err == nil && len(raw) > 0 {
			var w wishlist.Wishlist
			if err := json.Unmarshal(raw, &w); err == nil {
				return &w, nil
			}
		}
	}

	w, err := s.repo.GetByUserWithItems(ctx, userID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		// return empty wishlist shape (no DB row yet)
		w = &wishlist.Wishlist{UserID: userID, Items: []wishlist.WishlistItem{}}
	}

	if s.redis != nil {
		if raw, err := json.Marshal(w); err == nil {
			_ = s.redis.Set(ctx, s.key(userID), raw, ttl).Err()
		}
	}

	return w, nil
}

func (s *Service) Add(ctx context.Context, userID, productID, variantID int64) error {
	if userID <= 0 || productID <= 0 || variantID <= 0 {
		return wishlist.ErrInvalidInput
	}
	if err := s.repo.AddItem(ctx, userID, productID, variantID); err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}

func (s *Service) Remove(ctx context.Context, userID, productID, variantID int64) error {
	if userID <= 0 || productID <= 0 || variantID <= 0 {
		return wishlist.ErrInvalidInput
	}
	if err := s.repo.RemoveItem(ctx, userID, productID, variantID); err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}

func (s *Service) Clear(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return wishlist.ErrInvalidInput
	}
	if err := s.repo.Clear(ctx, userID); err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}