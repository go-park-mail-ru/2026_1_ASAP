package profile

import (
	"context"
	"fmt"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProfileAdapter struct {
	client profilev1.ProfileClient
}

func New(c profilev1.ProfileClient) *ProfileAdapter {
	return &ProfileAdapter{client: c}
}

func (p *ProfileAdapter) GetUserByID(ctx context.Context, id int64) (*pdomain.Profile, error) {
	resp, err := p.client.GetProfile(ctx, &profilev1.RequestGetProfile{UserId: id})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, fmt.Errorf("profile grpc GetProfile: %w", pdomain.ErrNotFound)
		}
		return nil, fmt.Errorf("profile grpc GetProfile: %w", err)
	}

	var avatarPtr *string
	if a := resp.GetAvatar(); a != "" {
		copy := a
		avatarPtr = &copy
	}

	var lastName *string
	if s := resp.GetLastName(); s != "" {
		copy := s
		lastName = &copy
	}

	return &pdomain.Profile{
		UserId:    resp.GetUserId(),
		FirstName: resp.GetFirstName(),
		LastName:  lastName,
		Avatar:    avatarPtr,
	}, nil
}

type ContactSnapshot struct {
	ContactUserID    int64
	FirstName        string
	LastName         *string
	ContactAvatarURL *string
}

func (p *ProfileAdapter) HasContact(ctx context.Context, userID, contactUserID int64) (bool, error) {
	resp, err := p.client.HasContact(ctx, &profilev1.RequestHasContact{
		UserId:        userID,
		ContactUserId: contactUserID,
	})
	if err != nil {
		return false, fmt.Errorf("profile grpc HasContact: %w", err)
	}
	return resp.GetExists(), nil
}

func (p *ProfileAdapter) GetContact(ctx context.Context, userID, contactUserID int64) (*ContactSnapshot, error) {
	resp, err := p.client.GetContact(ctx, &profilev1.RequestGetContact{
		UserId:        userID,
		ContactUserId: contactUserID,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, fmt.Errorf("profile grpc GetContact: %w", pdomain.ErrNotFound)
		}
		return nil, fmt.Errorf("profile grpc GetContact: %w", err)
	}
	item := resp.GetContact()
	if item == nil {
		return nil, fmt.Errorf("profile grpc GetContact: %w", pdomain.ErrNotFound)
	}
	var lastName *string
	if item.LastName != nil {
		ln := item.GetLastName()
		lastName = &ln
	}
	var avatar *string
	if item.ContactAvatarUrl != nil {
		u := item.GetContactAvatarUrl()
		avatar = &u
	}
	return &ContactSnapshot{
		ContactUserID:    item.GetContactUserId(),
		FirstName:        item.GetFirstName(),
		LastName:         lastName,
		ContactAvatarURL: avatar,
	}, nil
}

func (p *ProfileAdapter) UpdateLastSeen(ctx context.Context, userID int64) error {
	_, err := p.client.UpdateLastSeen(ctx, &profilev1.RequestUpdateLastSeen{UserId: userID})
	if err != nil {
		return fmt.Errorf("profile grpc UpdateLastSeen: %w", err)
	}
	return nil
}
