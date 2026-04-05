package contacts

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
	mock "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/contacts/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestPositiveContactsService_GetContacts(t *testing.T) {
	time := time.Now().UTC().Truncate(time.Second)

	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
	}

	type args struct {
		ctx context.Context
		userID int64
	}

	tests := []struct{
		name string
		prepare func(*fields)
		args args
		want []*dto.ContactResponse
	}{
		{
			name: "Succesful get contacts",
			prepare: func(f *fields) {
				f.contactRepository.EXPECT().GetAllContactsByUserID(context.Background(), int64(123)).Return([]*domain.Contact{
					&domain.Contact{
						UserID: 123, ContactName: "danil_kolbasenko", ContactUserID: 124, ContactAvatarUrl: nil, CreatedAt: time, UpdatedAt: time,
					},
					&domain.Contact{
						UserID: 124, ContactName: "daniil_kolbasenko", ContactUserID: 123, ContactAvatarUrl: nil, CreatedAt: time, UpdatedAt: time,
					},
				} , nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: []*dto.ContactResponse{
				&dto.ContactResponse{
					UserID: 123,
					ContactUserID: 124,
					ContactName: "danil_kolbasenko",
					ContactAvatarUrl: nil,
					CreatedAt: time,
				},
				&dto.ContactResponse{
					UserID: 124,
					ContactUserID: 123,
					ContactName: "daniil_kolbasenko",
					ContactAvatarUrl: nil,
					CreatedAt: time,
				},
			},
		},
		{
			name: "Successful get empty contacts",
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
		ctx context.Context
		userID int64
	}

	tests := []struct{
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "Failed get contacts",
			prepare: func(f *fields) {
				f.contactRepository.EXPECT().GetAllContactsByUserID(context.Background(), int64(123)).Return(([]*domain.Contact)(nil), errors.New("failed to get contacts"))
			},
			args: args{ctx: context.Background(), userID: 123},
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
		userRepository *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx context.Context
		contactRequest dto.AddContactRequest
		userID int64
	}

	tests := []struct{
		name string
		prepare func(f *fields)
		args args
		want *dto.ContactResponse
	}{
		{
			name: "Success add contact",
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
                        if contact.ContactName != "danil_kolbasenko" {
                            return nil, errors.New("wrong ContactName")
                        }
                        return &domain.Contact{
                            UserID:           123,
                            ContactName:      "danil_kolbasenko",
                            ContactUserID:    124,
                            ContactAvatarUrl: nil,
                            CreatedAt:        fixedTime,
                            UpdatedAt:        fixedTime,
                        }, nil
                    })
			},
			args: func() args {
				return args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, ContactName: "danil_kolbasenko"}, userID: 123}
			}(),
			want: &dto.ContactResponse{
				UserID: 123,
				ContactUserID: 124,
				ContactName: "danil_kolbasenko",
				ContactAvatarUrl: nil,
				CreatedAt: fixedTime,
			},
		},
		{
			name: "Success add contact with empty contact name",
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
                        if contact.ContactName != "" {
                            return nil, errors.New("wrong ContactName")
                        }
                        return &domain.Contact{
                            UserID:           123,
                            ContactName:      "",
                            ContactUserID:    124,
                            ContactAvatarUrl: nil,
                            CreatedAt:        fixedTime,
                            UpdatedAt:        fixedTime,
                        }, nil
                    })
			},
			args: func() args {
				return args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, ContactName: ""}, userID: 123}
			}(),
			want: &dto.ContactResponse{
				UserID: 123,
				ContactUserID: 124,
				ContactName: "",
				ContactAvatarUrl: nil,
				CreatedAt: fixedTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository: mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo: f.userRepository,
			}
			result, err := s.AddContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeContactService_AddContact(t *testing.T) {
	//fixedTime := time.Now().UTC().Truncate(time.Second)

	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx context.Context
		contactRequest dto.AddContactRequest
		userID int64
	}

	tests := []struct{
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "failed add contact: user not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return((*domainUser.User)(nil), domainUser.ErrNotFound)
				
			},
			args: args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, ContactName: "danil_kolbasenko"}, userID: 123},
			wantErr: domainUser.ErrNotFound,
		},
		{
			name: "failed add contact: contact already exists",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
			},
			args: args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, ContactName: "danil_kolbasenko"}, userID: 123},
			wantErr: domain.ErrContactExists,
		},
		{
			name: "failed add contact: unknown error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)
				f.contactRepository.EXPECT().CreateContact(context.Background(), gomock.Any()).
                    DoAndReturn(func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
                        return nil, errors.New("Uknown error")
                    })
			},
			args: args{ctx: context.Background(), contactRequest: dto.AddContactRequest{ContactUserID: 124, ContactName: "danil_kolbasenko"}, userID: 123},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository: mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo: f.userRepository,
			}
			result, err := s.AddContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestPositiveContactService_DeleteContact(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx context.Context
		contactRequest dto.DeleteContactRequest
		userID int64
	}

	contact := &dto.DeleteContactRequest{
		ContactUserID: 124,
	}

	tests := []struct{
		name string
		prepare func(f *fields)
		args args
		wantErr error
	}{
		{
			name: "success delete contact",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
				f.contactRepository.EXPECT().DeleteContact(context.Background(), int64(123), int64(124)).Return(nil)
			},
			args: args{ctx: context.Background(), contactRequest: *contact, userID: 123},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository: mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo: f.userRepository,
			}
			err := s.DeleteContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			require.Nil(t, err)
			require.NoError(t, err)
			require.Equal(t, tt.wantErr, err)
		})
	}
}

func TestNegativeContactService_DeleteContact(t *testing.T) {
	type fields struct {
		contactRepository *mock.MockContactRepositoryInterface
		userRepository *mock.MockUserRepositoryInterface
	}

	type args struct {
		ctx context.Context
		contactRequest dto.DeleteContactRequest
		userID int64
	}

	contact := &dto.DeleteContactRequest{
		ContactUserID: 124,
	}

	tests := []struct{
		name string
		prepare func(f *fields)
		args args
		wantAnyErr bool
		wantErr error
	}{
		{
			name: "failed to delete contact: user not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return((*domainUser.User)(nil), domainUser.ErrNotFound)
			},
			args: args{ctx: context.Background(), contactRequest: *contact, userID: 123},
			wantErr: domainUser.ErrNotFound,
		},
		{
			name: "failed to delete contact: contact not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(false, nil)
			},
			args: args{ctx: context.Background(), contactRequest: *contact, userID: 123},
			wantErr: domain.ErrContactNotFound,
		},
		{
			name: "failed to delete contact: unknown error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(context.Background(), int64(124)).Return(&domainUser.User{Id: 124, Login: "danil_kolbasenko"}, nil)
				f.contactRepository.EXPECT().IsContact(context.Background(), int64(123), int64(124)).Return(true, nil)
				f.contactRepository.EXPECT().DeleteContact(context.Background(), int64(123), int64(124)).Return(errors.New("Unknown error"))
			},
			args: args{ctx: context.Background(), contactRequest: *contact, userID: 123},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				contactRepository: mock.NewMockContactRepositoryInterface(ctrl),
				userRepository: mock.NewMockUserRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ContactService{
				contactRepo: f.contactRepository,
				userRepo: f.userRepository,
			}
			err := s.DeleteContact(tt.args.ctx, tt.args.contactRequest, tt.args.userID)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}