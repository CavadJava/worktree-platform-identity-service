package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"shop-service/internal/models"
	"shop-service/internal/repository"
	"shop-service/internal/roles"
)

var (
	ErrApplicationNotFound       = repository.ErrApplicationNotFound
	ErrApplicationAlreadyDecided = errors.New("application already decided")
	ErrApplicationNotFormSent    = errors.New("the form must be sent before the application can be approved")
	ErrApplicationNameMissing    = errors.New("name is required")
)

const defaultTemplateVersion = "v1"

type notifier interface {
	Send(ctx context.Context, notificationType, email, fullName, message string) error
}

type contactLookup interface {
	GetContact(ctx context.Context, userID string) (*repository.Contact, error)
}

type ShopApplicationService struct {
	appRepo  *repository.ShopApplicationRepository
	shopRepo *repository.ShopRepository
	users    *repository.UserAssignmentRepository
	contacts contactLookup
	notifier notifier
}

func NewShopApplicationService(
	appRepo *repository.ShopApplicationRepository,
	shopRepo *repository.ShopRepository,
	users *repository.UserAssignmentRepository,
	notifier notifier,
) *ShopApplicationService {
	return &ShopApplicationService{appRepo: appRepo, shopRepo: shopRepo, users: users, contacts: users, notifier: notifier}
}

func (s *ShopApplicationService) Submit(ctx context.Context, applicantID, name, description string) (*models.ShopApplication, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrApplicationNameMissing
	}

	now := time.Now().UTC()
	app := &models.ShopApplication{
		ID:              uuid.NewString(),
		ApplicantID:     applicantID,
		Name:            name,
		Description:     description,
		TemplateVersion: defaultTemplateVersion,
		Status:          models.ApplicationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *ShopApplicationService) Get(ctx context.Context, id string) (*models.ShopApplication, error) {
	return s.appRepo.FindByID(ctx, id)
}

func (s *ShopApplicationService) List(ctx context.Context, status string) ([]*models.ShopApplication, error) {
	return s.appRepo.List(ctx, status)
}

// SendForm is the administrator's first decision: instead of rejecting
// outright, they let the applicant proceed. A temporary shop is created
// right away and the applicant becomes its admin(4) so they can fill in the
// real details themselves via the normal shop PUT endpoint — the temporary
// shop *is* the form. It stays hidden from public listings until approved.
func (s *ShopApplicationService) SendForm(ctx context.Context, reviewerID, applicationID string) (*models.ShopApplication, error) {
	app, err := s.appRepo.FindByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if app.Status != models.ApplicationStatusPending {
		return nil, ErrApplicationAlreadyDecided
	}

	shop, err := s.createShop(ctx, app.ApplicantID, app.Name, app.Description, true)
	if err != nil {
		return nil, err
	}

	if err := s.users.AssignShopAdmin(ctx, app.ApplicantID, shop.ID, roles.ShopLevelAdmin); err != nil {
		return nil, err
	}

	updated, err := s.appRepo.Decide(ctx, applicationID, models.ApplicationStatusFormSent, reviewerID, "", &shop.ID)
	if err != nil {
		return nil, err
	}

	s.notify(ctx, app.ApplicantID, "shop_application_form_sent",
		"Mağaza müraciətiniz nəzərdən keçirildi — indi mağaza məlumatlarınızı doldura bilərsiniz.")
	return updated, nil
}

// Approve finalizes an application that already went through SendForm: the
// temporary shop becomes permanent (visible in public listings).
func (s *ShopApplicationService) Approve(ctx context.Context, reviewerID, applicationID string) (*models.ShopApplication, error) {
	app, err := s.appRepo.FindByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if app.Status != models.ApplicationStatusFormSent || app.ShopID == nil {
		return nil, ErrApplicationNotFormSent
	}

	if err := s.shopRepo.SetTemporary(ctx, *app.ShopID, false); err != nil {
		return nil, err
	}

	updated, err := s.appRepo.Decide(ctx, applicationID, models.ApplicationStatusApproved, reviewerID, "", app.ShopID)
	if err != nil {
		return nil, err
	}

	s.notify(ctx, app.ApplicantID, "shop_application_approved", "Mağazanız təsdiqləndi və artıq aktivdir.")
	return updated, nil
}

// Reject can happen at either stage: outright (still "pending", no shop was
// ever created) or after SendForm (the temporary shop and the applicant's
// admin(4) assignment are both torn down).
func (s *ShopApplicationService) Reject(ctx context.Context, reviewerID, applicationID, reason string) (*models.ShopApplication, error) {
	app, err := s.appRepo.FindByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if app.Status != models.ApplicationStatusPending && app.Status != models.ApplicationStatusFormSent {
		return nil, ErrApplicationAlreadyDecided
	}

	if app.Status == models.ApplicationStatusFormSent && app.ShopID != nil {
		if err := s.users.RevokeShopAssignment(ctx, app.ApplicantID); err != nil {
			return nil, err
		}
		if err := s.shopRepo.Delete(ctx, *app.ShopID); err != nil {
			return nil, err
		}
	}

	updated, err := s.appRepo.Decide(ctx, applicationID, models.ApplicationStatusRejected, reviewerID, reason, nil)
	if err != nil {
		return nil, err
	}

	s.notify(ctx, app.ApplicantID, "shop_application_rejected", reason)
	return updated, nil
}

func (s *ShopApplicationService) createShop(ctx context.Context, ownerID, name, description string, temporary bool) (*models.Shop, error) {
	now := time.Now().UTC()
	shop := &models.Shop{
		ID:          uuid.NewString(),
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		Temporary:   temporary,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.shopRepo.Create(ctx, shop); err != nil {
		return nil, err
	}
	return shop, nil
}

// notify best-efforts a status-change message to the applicant — a failure
// here never undoes the decision that was already committed to the DB.
func (s *ShopApplicationService) notify(ctx context.Context, applicantID, notificationType, message string) {
	contact, err := s.contacts.GetContact(ctx, applicantID)
	if err != nil {
		log.Printf("warning: failed to look up contact for %s: %v", applicantID, err)
		return
	}
	if err := s.notifier.Send(ctx, notificationType, contact.Email, contact.FullName, message); err != nil {
		log.Printf("warning: failed to send %s notification to %s: %v", notificationType, contact.Email, err)
	}
}
