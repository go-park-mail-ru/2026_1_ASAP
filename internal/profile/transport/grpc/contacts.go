package grpc

import (
	"context"
	"errors"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	contactdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/profile"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (p ProfileServer) ListContacts(ctx context.Context, req *profilev1.RequestListContacts) (*profilev1.ResponseListContacts, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if p.contactService == nil {
		return nil, status.Error(codes.Unimplemented, "contacts not configured")
	}
	out, err := p.contactService.GetContacts(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "contacts internal error")
	}
	items := make([]*profilev1.ContactItem, 0, len(out))
	for _, c := range out {
		items = append(items, contactResponseToProto(c))
	}
	return &profilev1.ResponseListContacts{Contacts: items}, nil
}

func (p ProfileServer) AddContact(ctx context.Context, req *profilev1.RequestAddContact) (*profilev1.ResponseAddContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, status.Error(codes.Unimplemented, "contacts not configured")
	}
	d := dto.AddContactRequest{
		ContactUserID: req.GetContactUserId(),
		FirstName:     req.GetFirstName(),
	}
	if req.LastName != nil {
		ln := req.GetLastName()
		d.LastName = &ln
	}
	created, err := p.contactService.AddContact(ctx, d, req.GetUserId())
	if err != nil {
		return nil, mapContactError(err)
	}
	return &profilev1.ResponseAddContact{Contact: contactResponseToProto(created)}, nil
}

func (p ProfileServer) DeleteContact(ctx context.Context, req *profilev1.RequestDeleteContact) (*profilev1.ResponseDeleteContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, status.Error(codes.Unimplemented, "contacts not configured")
	}
	err := p.contactService.DeleteContact(ctx, dto.DeleteContactRequest{ContactUserID: req.GetContactUserId()}, req.GetUserId())
	if err != nil {
		return nil, mapContactError(err)
	}
	return &profilev1.ResponseDeleteContact{}, nil
}

func mapContactError(err error) error {
	switch {
	case errors.Is(err, contactdomain.ErrCantCreateContactWithYourself):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, contactdomain.ErrContactExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, contactdomain.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pdomain.ErrNotFound):
		return status.Error(codes.NotFound, "user not found")
	default:
		return status.Error(codes.Internal, "contacts internal error")
	}
}

func contactResponseToProto(c *dto.ContactResponse) *profilev1.ContactItem {
	if c == nil {
		return nil
	}
	item := &profilev1.ContactItem{
		UserId:        c.UserID,
		ContactUserId: c.ContactUserID,
		FirstName:     c.FirstName,
		CreatedAt:     timestamppb.New(c.CreatedAt),
	}
	if c.LastName != nil {
		ln := *c.LastName
		item.LastName = &ln
	}
	if c.ContactAvatarUrl != nil {
		u := *c.ContactAvatarUrl
		item.ContactAvatarUrl = &u
	}
	return item
}
