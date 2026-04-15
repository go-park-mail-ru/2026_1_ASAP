package contacts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
	mock "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/contacts/mock"
)

func TestPositiveContactsService_GetContacts(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    []*dto.ContactResponse
	}{
		{
			name: "successful get contacts",
			prepare: func(f *fields) {
				f.contactRepository.EXPECT().GetAllContactsByUserID(context.Background(), int64(123)).Return([]*domain.Contact{
					{
						UserID: 123, FirstName: "danil_kolbasenko", LastName: nil, ContactUserID: 124,
						ContactAvatarUrl: nil, CreatedAt: now, UpdatedAt: now,
					},
					{
						UserID: 124, FirstName: "daniil_kolbasenko", LastName: nil, ContactUserID: 123,
						ContactAvatarUrl: nil, CreatedAt: now, UpdatedAt: now,
					},
				}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: []*dto.ContactResponse{
				{
					UserID: 123, ContactUserID: 124,
					FirstName: "danil_kolbasenko", LastName: nil,
					ContactAvatarUrl: nil, CreatedAt: now,
				},
				{
					UserID: 124, ContactUserID: 123,
					FirstName: "daniil_kolbasenko", LastName: nil,
					ContactAvatarUrl: nil, CreatedAt: now,
				},
			},
		},
		{
			name: "successful get empty contacts",
			prepare: func(f *fields) {
				f.contactRepository.EXPECT().GetAllContactsByUserID(context.Background(), int64(123)).Return([]*domain.Contact{}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: []*dto.ContactResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
			}
			result, err := s.GetContacts(tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeContactsService_GetContacts(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantAnyErr bool
	}{
		{
			name: "failed get contacts",
			prepare: func(f *fields) {
				f.contactRepository.EXPECT().GetAllContactsByUserID(context.Background(), int64(123)).
					Return(([]*domain.Contact)(nil), errors.New("failed to get contacts"))
			},
			args:       args{ctx: context.Background(), userID: 123},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
			}
			result, err := s.GetContacts(tt.args.ctx, tt.args.userID)
			require.Nil(t, result)
			require.Error(t, err)
		})
	}
}

func TestPositiveContactsService_AddContact(t *testing.T) {
	fixedTime := time.Now().UTC().Truncate(time.Second)

	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository    *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx            context.Context
		contactRequest dto.AddContactRequest
		userID         int64
	}

	tests := []struct {
		prepare func(f *fields)
		want    *dto.ContactResponse
		name    string
		args    args
	}{
		{
			name: "success add contact",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)

				f.contactRepository.EXPECT().CreateContact(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
						if contact.UserID != 123 {
							return nil, errors.New("wrong UserID")
						}
						if contact.ContactUserID != 124 {
							return nil, errors.New("wrong ContactUserID")
						}
						if contact.FirstName != "danil_kolbasenko" {
							return nil, errors.New("wrong FirstName")
						}
						return &domain.Contact{
							UserID:           123,
							FirstName:        "danil_kolbasenko",
							LastName:         nil,
							ContactUserID:    124,
							ContactAvatarUrl: nil,
							CreatedAt:        fixedTime,
							UpdatedAt:        fixedTime,
						}, nil
					})
			},
			args: args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, FirstName: "danil_kolbasenko"}, userID: 123},
			want: &dto.ContactResponse{
				UserID:           123,
				ContactUserID:    124,
				FirstName:        "danil_kolbasenko",
				LastName:         nil,
				ContactAvatarUrl: nil,
				CreatedAt:        fixedTime,
			},
		},
		{
			name: "success add contact with empty first name uses login",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)

				f.contactRepository.EXPECT().CreateContact(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
						if contact.UserID != 123 {
							return nil, errors.New("wrong UserID")
						}
						if contact.ContactUserID != 124 {
							return nil, errors.New("wrong ContactUserID")
						}
						if contact.FirstName != "danil_kolbasenko" {
							return nil, errors.New("wrong FirstName: expected login fallback")
						}
						return &domain.Contact{
							UserID:           123,
							FirstName:        "danil_kolbasenko",
							LastName:         nil,
							ContactUserID:    124,
							ContactAvatarUrl: nil,
							CreatedAt:        fixedTime,
							UpdatedAt:        fixedTime,
						}, nil
					})
			},
			args: args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, FirstName: ""}, userID: 123},
			want: &dto.ContactResponse{
				UserID:           123,
				ContactUserID:    124,
				FirstName:        "danil_kolbasenko",
				LastName:         nil,
				ContactAvatarUrl: nil,
				CreatedAt:        fixedTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository:    mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo:    f.userRepository,
			}
			result, err := s.AddContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeContactService_AddContact(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository    *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx            context.Context
		contactRequest dto.AddContactRequest
		userID         int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "failed add contact: user not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return((*domainUser.User)(nil), domainUser.ErrNotFound)
			},
			args:    args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, FirstName: "danil_kolbasenko"}, userID: 123},
			wantErr: domainUser.ErrNotFound,
		},
		{
			name: "failed add contact: contact already exists",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
			},
			args:    args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, FirstName: "danil_kolbasenko"}, userID: 123},
			wantErr: domain.ErrContactExists,
		},
		{
			name: "failed add contact: unknown error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)
				f.contactRepository.EXPECT().CreateContact(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
						return nil, errors.New("unknown error")
					})
			},
			args:       args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, FirstName: "danil_kolbasenko"}, userID: 123},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository:    mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo:    f.userRepository,
			}
			result, err := s.AddContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestPositiveContactService_DeleteContact(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository    *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx            context.Context
		contactRequest dto.DeleteContactRequest
		userID         int64
	}

	contact := dto.DeleteContactRequest{ContactUserID: 124}

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
	}{
		{
			name: "success delete contact",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
				f.contactRepository.EXPECT().DeleteContact(context.Background(), int64(123), int64(124)).Return(nil)
			},
			args: args{ctx: context.Background(), contactRequest: contact, userID: 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository:    mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo:    f.userRepository,
			}
			err := s.DeleteContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.NoError(t, err)
		})
	}
}

func TestNegativeContactService_DeleteContact(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository    *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx            context.Context
		contactRequest dto.DeleteContactRequest
		userID         int64
	}

	contact := dto.DeleteContactRequest{ContactUserID: 124}

	tests := []struct {
		wantErr    error
		prepare    func(f *fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "failed to delete contact: user not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return((*domainUser.User)(nil), domainUser.ErrNotFound)
			},
			args:    args{ctx: context.Background(), contactRequest: contact, userID: 123},
			wantErr: domainUser.ErrNotFound,
		},
		{
			name: "failed to delete contact: contact not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), contactRequest: contact, userID: 123},
			wantErr: domain.ErrContactNotFound,
		},
		{
			name: "failed to delete contact: unknown error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
				f.contactRepository.EXPECT().DeleteContact(context.Background(), int64(123), int64(124)).Return(errors.New("unknown error"))
			},
			args:       args{ctx: context.Background(), contactRequest: contact, userID: 123},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository:    mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo:    f.userRepository,
			}
			err := s.DeleteContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
