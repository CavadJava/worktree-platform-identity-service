package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"user-service/internal/models"
	"user-service/internal/repository"
)

var (
	ErrAddressNotFound  = repository.ErrAddressNotFound
	ErrAddressForbidden = errors.New("address belongs to another user")
)

type AddressService struct {
	repo *repository.AddressRepository
}

func NewAddressService(repo *repository.AddressRepository) *AddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) Create(ctx context.Context, userID, title, fullAddress, city, phone string, isDefault bool) (*models.Address, error) {
	if isDefault {
		if err := s.repo.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	a := &models.Address{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       strings.TrimSpace(title),
		FullAddress: strings.TrimSpace(fullAddress),
		City:        strings.TrimSpace(city),
		Phone:       strings.TrimSpace(phone),
		IsDefault:   isDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AddressService) List(ctx context.Context, userID string) ([]*models.Address, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *AddressService) Update(ctx context.Context, userID, id, title, fullAddress, city, phone string, isDefault bool) (*models.Address, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.UserID != userID {
		return nil, ErrAddressForbidden
	}
	if isDefault && !existing.IsDefault {
		if err := s.repo.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.repo.Update(ctx, id, strings.TrimSpace(title), strings.TrimSpace(fullAddress), strings.TrimSpace(city), strings.TrimSpace(phone), isDefault)
}

func (s *AddressService) Delete(ctx context.Context, userID, id string) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return ErrAddressForbidden
	}
	return s.repo.Delete(ctx, id)
}
