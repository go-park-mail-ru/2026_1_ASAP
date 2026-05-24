package grpc

import (
	"context"
	"errors"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	contactdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (p ProfileServer) ListContacts(ctx context.Context, req *profilev1.RequestListContacts) (*profilev1.ResponseListContacts, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}
	if p.contactService == nil {
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts not configured")
	}
	out, err := p.contactService.GetContacts(ctx, req.GetUserId())
	if err != nil {
		p.Log(ctx).Error("failed to list contacts", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts internal error")
	}
	items := make([]*profilev1.ContactItem, 0, len(out))
	for _, c := range out {
		items = append(items, contactResponseToProto(c))
	}
	return &profilev1.ResponseListContacts{Contacts: items}, nil
}

func (p ProfileServer) AddContact(ctx context.Context, req *profilev1.RequestAddContact) (*profilev1.ResponseAddContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts not configured")
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
		p.Log(ctx).Info("failed to add contact", zap.Int64("user_id", req.GetUserId()), zap.Int64("contact_user_id", req.GetContactUserId()), zap.Error(err))
		return nil, mapContactError(err)
	}
	return &profilev1.ResponseAddContact{Contact: contactResponseToProto(created)}, nil
}

func (p ProfileServer) DeleteContact(ctx context.Context, req *profilev1.RequestDeleteContact) (*profilev1.ResponseDeleteContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts not configured")
	}
	err := p.contactService.DeleteContact(ctx, dto.DeleteContactRequest{ContactUserID: req.GetContactUserId()}, req.GetUserId())
	if err != nil {
		p.Log(ctx).Info("failed to delete contact", zap.Int64("user_id", req.GetUserId()), zap.Int64("contact_user_id", req.GetContactUserId()), zap.Error(err))
		return nil, mapContactError(err)
	}
	return &profilev1.ResponseDeleteContact{}, nil
}

func (p ProfileServer) HasContact(ctx context.Context, req *profilev1.RequestHasContact) (*profilev1.ResponseHasContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts not configured")
	}
	exists, err := p.contactService.HasContact(ctx, req.GetUserId(), req.GetContactUserId())
	if err != nil {
		p.Log(ctx).Error("failed to check contact", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts internal error")
	}
	return &profilev1.ResponseHasContact{Exists: exists}, nil
}

func (p ProfileServer) GetContact(ctx context.Context, req *profilev1.RequestGetContact) (*profilev1.ResponseGetContact, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetContactUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id and contact_user_id are required")
	}
	if p.contactService == nil {
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts not configured")
	}
	contact, err := p.contactService.GetContact(ctx, req.GetUserId(), req.GetContactUserId())
	if err != nil {
		p.Log(ctx).Info("failed to get contact", zap.Int64("user_id", req.GetUserId()), zap.Int64("contact_user_id", req.GetContactUserId()), zap.Error(err))
		return nil, mapContactError(err)
	}
	return &profilev1.ResponseGetContact{Contact: contactResponseToProto(contact)}, nil
}

func mapContactError(err error) error {
	switch {
	case errors.Is(err, contactdomain.ErrCantCreateContactWithYourself):
		return grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_SELF), "cannot add yourself as a contact")
	case errors.Is(err, contactdomain.ErrContactExists):
		return grpcerr.New(codes.AlreadyExists, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_ALREADY_EXISTS), "contact already exists")
	case errors.Is(err, contactdomain.ErrContactNotFound):
		return grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_NOT_FOUND), "contact not found")
	case errors.Is(err, pdomain.ErrNotFound):
		return grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "user not found")
	default:
		return grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "contacts internal error")
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
		IsOnline:      c.IsOnline,
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
