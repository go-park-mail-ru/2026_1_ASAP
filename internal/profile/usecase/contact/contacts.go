package contacts

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/profile"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=contacts.go -destination=mock/contacts_mock.go -package=mock
type ContactRepositoryInterface interface {
	GetAllContactsByUserID(ctx context.Context, userID int64) ([]*domain.Contact, error)
	CreateContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error)
	DeleteContact(ctx context.Context, userID, contactUserID int64) error
	IsContact(ctx context.Context, userID, contactUserID int64) (bool, error)
}

type ProfileRepositoryInterface interface {
	GetProfileById(ctx context.Context, id int64) (*pdomain.Profile, error)
}

type ContactService struct {
	contactRepo ContactRepositoryInterface
	profileRepo ProfileRepositoryInterface
}

func NewContactService(contactRepo ContactRepositoryInterface, userRepo ProfileRepositoryInterface) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		profileRepo: userRepo,
	}
}

func (s *ContactService) GetContacts(ctx context.Context, userID int64) ([]*dto.ContactResponse, error) {
	contacts, err := s.contactRepo.GetAllContactsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	if len(contacts) == 0 {
		return []*dto.ContactResponse{}, nil
	}

	result := make([]*dto.ContactResponse, 0, len(contacts))
	for _, contact := range contacts {
		result = append(result, &dto.ContactResponse{
			UserID:           contact.UserID,
			ContactUserID:    contact.ContactUserID,
			ContactAvatarUrl: contact.ContactAvatarUrl,
			FirstName:        contact.FirstName,
			LastName:         contact.LastName,
			CreatedAt:        contact.CreatedAt,
		})
	}

	return result, nil
}

func (s *ContactService) AddContact(ctx context.Context, contactRequest dto.AddContactRequest, userID int64) (*dto.ContactResponse, error) {
	if userID == contactRequest.ContactUserID {
		return nil, domain.ErrCantCreateContactWithYourself
	}

	contactUser, err := s.profileRepo.GetProfileById(ctx, contactRequest.ContactUserID)
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			return nil, pdomain.ErrNotFound
		}

		return nil, fmt.Errorf("failed to check contact user id: %w", err)
	}

	if contactRequest.FirstName == "" {
		contactRequest.FirstName = contactUser.FirstName
	}

	exists, err := s.contactRepo.IsContact(ctx, userID, contactRequest.ContactUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check your if this is your contact: %w", err)
	}
	if exists {
		return nil, domain.ErrContactExists
	}

	contact := &domain.Contact{
		UserID:        userID,
		ContactUserID: contactRequest.ContactUserID,
		FirstName:     contactRequest.FirstName,
		LastName:      contactRequest.LastName,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	result, err := s.contactRepo.CreateContact(ctx, contact)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	contactDTO := &dto.ContactResponse{
		UserID:           result.UserID,
		ContactUserID:    result.ContactUserID,
		FirstName:        result.FirstName,
		LastName:         result.LastName,
		ContactAvatarUrl: result.ContactAvatarUrl,
		CreatedAt:        result.CreatedAt,
	}

	return contactDTO, nil
}

func (s *ContactService) DeleteContact(ctx context.Context, contactRequest dto.DeleteContactRequest, userID int64) error {
	_, err := s.profileRepo.GetProfileById(ctx, contactRequest.ContactUserID)
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			return pdomain.ErrNotFound
		}

		return fmt.Errorf("failed to check contact user id: %w", err)
	}

	exists, err := s.contactRepo.IsContact(ctx, userID, contactRequest.ContactUserID)
	if err != nil {
		return fmt.Errorf("failed to check your if this is your contact: %w", err)
	}
	if !exists {
		return domain.ErrContactNotFound
	}

	contact := &domain.Contact{
		UserID:        userID,
		ContactUserID: contactRequest.ContactUserID,
	}

	err = s.contactRepo.DeleteContact(ctx, contact.UserID, contact.ContactUserID)
	if err != nil {
		if errors.Is(err, domain.ErrContactNotFound) {
			return domain.ErrContactNotFound
		}

		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}
