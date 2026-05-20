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

func (p *ProfileAdapter) UpdateLastSeen(ctx context.Context, userID int64) error {
	_, err := p.client.UpdateLastSeen(ctx, &profilev1.RequestUpdateLastSeen{UserId: userID})
	if err != nil {
		return fmt.Errorf("profile grpc UpdateLastSeen: %w", err)
	}
	return nil
}
