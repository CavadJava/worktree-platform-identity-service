package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"localization-service/internal/models"
	"localization-service/internal/repository"
)

var (
	ErrTranslationNotFound = repository.ErrTranslationNotFound
	ErrNamespaceRequired   = errors.New("namespace is required")
	ErrKeyRequired         = errors.New("key is required")
	ErrLocaleRequired      = errors.New("locale is required")
	ErrValueRequired       = errors.New("value is required")
)

type TranslationService struct {
	repo          *repository.TranslationRepository
	defaultLocale string
}

func NewTranslationService(repo *repository.TranslationRepository, defaultLocale string) *TranslationService {
	return &TranslationService{repo: repo, defaultLocale: defaultLocale}
}

func (s *TranslationService) Upsert(ctx context.Context, namespace, key, locale, value string) (*models.Translation, error) {
	namespace = strings.TrimSpace(namespace)
	key = strings.TrimSpace(key)
	locale = strings.TrimSpace(locale)

	if namespace == "" {
		return nil, ErrNamespaceRequired
	}
	if key == "" {
		return nil, ErrKeyRequired
	}
	if locale == "" {
		return nil, ErrLocaleRequired
	}
	if value == "" {
		return nil, ErrValueRequired
	}

	now := time.Now().UTC()
	t := &models.Translation{
		ID:        uuid.NewString(),
		Namespace: namespace,
		Key:       key,
		Locale:    locale,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Upsert(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get looks up one translation, falling back to the service's default
// locale if the requested locale doesn't have this key.
func (s *TranslationService) Get(ctx context.Context, namespace, key, locale string) (*models.Translation, error) {
	t, err := s.repo.Get(ctx, namespace, key, locale)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, repository.ErrTranslationNotFound) || locale == s.defaultLocale {
		return nil, err
	}
	return s.repo.Get(ctx, namespace, key, s.defaultLocale)
}

func (s *TranslationService) List(ctx context.Context, namespace, locale, key string) ([]*models.Translation, error) {
	return s.repo.List(ctx, namespace, locale, key)
}

// Map returns namespace+locale translations as a key->value map — the
// shape most convenient for a client to load once and use directly.
func (s *TranslationService) Map(ctx context.Context, namespace, locale string) (map[string]string, error) {
	translations, err := s.repo.List(ctx, namespace, locale, "")
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(translations))
	for _, t := range translations {
		result[t.Key] = t.Value
	}
	return result, nil
}

func (s *TranslationService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TranslationService) ListLocales(ctx context.Context) ([]string, error) {
	return s.repo.ListLocales(ctx)
}
