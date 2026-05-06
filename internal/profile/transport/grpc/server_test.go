//go:generate mockgen -destination=mock/profile_service_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/transport/grpc ProfileServiceInterface
//go:generate mockgen -destination=mock/contact_service_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/transport/grpc ContactServiceInterface
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	contactdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	profileDto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/transport/grpc/mock"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestPositiveProfileServer_GetProfile(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestGetProfile
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful get profile",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().GetUserProfile(gomock.Any(), int64(100)).Return(&profileDto.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
					LastName:  strPtr("Doe"),
					Bio:       strPtr("My bio"),
					BirthDate: strPtr("1990-01-01"),
					Avatar:    strPtr("avatar.jpg"),
					LastSeen:  timePtr(now),
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestGetProfile{UserId: 100},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				LastName:  strPtr("Doe"),
				Bio:       "My bio",
				BirthDate: "1990-01-01",
				Avatar:    "avatar.jpg",
				LastSeen:  timestamppb.New(now),
			},
		},
		{
			name: "Get profile without optional fields",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().GetUserProfile(gomock.Any(), int64(200)).Return(&profileDto.ResponseGetProfile{
					UserId:    200,
					FirstName: "Jane",
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestGetProfile{UserId: 200},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    200,
				FirstName: "Jane",
				Bio:       "",
				BirthDate: "",
				Avatar:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.GetProfile(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
			require.Equal(t, tt.want.GetFirstName(), resp.GetFirstName())
		})
	}
}

func TestNegativeProfileServer_GetProfile(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestGetProfile
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id (zero)",
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestGetProfile{UserId: 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Profile not found",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().GetUserProfile(gomock.Any(), int64(999)).Return(nil, pdomain.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestGetProfile{UserId: 999},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "UseCase error",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().GetUserProfile(gomock.Any(), int64(100)).Return(nil, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestGetProfile{UserId: 100},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			_, err := s.GetProfile(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveProfileServer_CreateProfile(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestCreateProfile
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful create profile",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().CreateProfile(gomock.Any(), &profileDto.RequestCreateProfile{
					ProfileID: 100,
					FirstName: "John",
				}).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestCreateProfile{
					ProfileId: 100,
					FirstName: "John",
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.CreateProfile(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestPositiveProfileServer_UpdateProfileAvatar(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateAvatar
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful update avatar",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatar(gomock.Any(), int64(100), gomock.Any()).Return(&profileDto.ResponseUpdateProfile{
					UserId:    100,
					FirstName: "John",
					Avatar:    strPtr("new_avatar.jpg"),
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{
					UserId:  100,
					Content: []byte("image data"),
					Type:    "image/jpeg",
				},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				Avatar:    "new_avatar.jpg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateProfileAvatar(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
			require.Equal(t, tt.want.GetAvatar(), resp.GetAvatar())
		})
	}
}

func TestNegativeProfileServer_UpdateProfileAvatar(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateAvatar
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id",
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 0, Content: []byte("data"), Type: "image/jpeg"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty content",
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 100, Content: []byte{}, Type: "image/jpeg"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Avatar too large",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatar(gomock.Any(), int64(100), gomock.Any()).Return(nil, pdomain.ErrAvatarTooLarge)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 100, Content: []byte("data"), Type: "image/jpeg"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid avatar type",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatar(gomock.Any(), int64(100), gomock.Any()).Return(nil, pdomain.ErrInvalidAvatarType)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 100, Content: []byte("data"), Type: "image/jpeg"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty avatar",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatar(gomock.Any(), int64(100), gomock.Any()).Return(nil, pdomain.ErrEmptyAvatar)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 100, Content: []byte("data"), Type: "image/jpeg"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Profile not found",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatar(gomock.Any(), int64(999), gomock.Any()).Return(nil, pdomain.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatar{UserId: 999, Content: []byte("data"), Type: "image/jpeg"},
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			_, err := s.UpdateProfileAvatar(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveProfileServer_UpdateProfileAvatarURL(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateAvatarURL
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful update avatar URL",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().UpdateProfileAvatarURL(gomock.Any(), int64(100), &profileDto.RequestUpdateAvatarURL{
					AvatarURL: "https://example.com/avatar.jpg",
				}).Return(&profileDto.ResponseUpdateProfile{
					UserId:    100,
					FirstName: "John",
					Avatar:    strPtr("https://example.com/avatar.jpg"),
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateAvatarURL{
					UserId:    100,
					AvatarUrl: "https://example.com/avatar.jpg",
				},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				Avatar:    "https://example.com/avatar.jpg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateProfileAvatarURL(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetAvatar(), resp.GetAvatar())
		})
	}
}

func TestPositiveProfileServer_UpdateProfileBio(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateBio
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful update bio",
			prepare: func(f *fields) {
				bio := "New bio"
				f.profileUseCase.EXPECT().UpdateProfileBio(gomock.Any(), int64(100), &profileDto.RequestUpdateBio{
					Bio: &bio,
				}).Return(&profileDto.ResponseUpdateProfile{
					UserId:    100,
					FirstName: "John",
					Bio:       &bio,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateBio{
					UserId: 100,
					Bio:    strPtr("New bio"),
				},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				Bio:       "New bio",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateProfileBio(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetBio(), resp.GetBio())
		})
	}
}

func TestPositiveProfileServer_UpdateProfileBirthDate(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateBirthDate
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful update birth date",
			prepare: func(f *fields) {
				birthDate := "1990-01-01"
				f.profileUseCase.EXPECT().UpdateProfileBirthDate(gomock.Any(), int64(100), &profileDto.RequestUpdateBirthDate{
					BirthDate: &birthDate,
				}).Return(&profileDto.ResponseUpdateProfile{
					UserId:    100,
					FirstName: "John",
					BirthDate: &birthDate,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateBirthDate{
					UserId:    100,
					BirthDate: strPtr("1990-01-01"),
				},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				BirthDate: "1990-01-01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateProfileBirthDate(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetBirthDate(), resp.GetBirthDate())
		})
	}
}

func TestPositiveProfileServer_UpdateProfileName(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestUpdateName
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseGetProfile
		name    string
		args    args
	}{
		{
			name: "Successful update name",
			prepare: func(f *fields) {
				lastName := "Doe"
				f.profileUseCase.EXPECT().UpdateProfileName(gomock.Any(), int64(100), &profileDto.RequestUpdateName{
					FirstName: "John",
					LastName:  &lastName,
				}).Return(&profileDto.ResponseUpdateProfile{
					UserId:    100,
					FirstName: "John",
					LastName:  &lastName,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestUpdateName{
					UserId:     100,
					FirstName:  "John",
					SecondName: strPtr("Doe"),
				},
			},
			want: &profilev1.ResponseGetProfile{
				UserId:    100,
				FirstName: "John",
				LastName:  strPtr("Doe"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateProfileName(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetFirstName(), resp.GetFirstName())
		})
	}
}

func TestPositiveProfileServer_SearchIdByLogin(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestSearchIdByLogin
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseSearchIdByLogin
		name    string
		args    args
	}{
		{
			name: "Successful search by login",
			prepare: func(f *fields) {
				f.profileUseCase.EXPECT().SearchIdByLogin(gomock.Any(), &profileDto.RequestSearchIdByLogin{
					Login: "john_doe",
				}).Return(&profileDto.ResponseSearchIdByLogin{
					UserId: 100,
					Login:  "john_doe",
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestSearchIdByLogin{Login: "john_doe"},
			},
			want: &profilev1.ResponseSearchIdByLogin{
				UserId: 100,
				Login:  "john_doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.SearchIdByLogin(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
		})
	}
}

func TestPositiveProfileServer_ListContacts(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestListContacts
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseListContacts
		name    string
		args    args
	}{
		{
			name: "Successful list contacts",
			prepare: func(f *fields) {
				f.contactService.EXPECT().GetContacts(gomock.Any(), int64(100)).Return([]*dto.ContactResponse{
					{
						UserID:           100,
						ContactUserID:    101,
						FirstName:        "John",
						LastName:         strPtr("Doe"),
						ContactAvatarUrl: strPtr("avatar.jpg"),
						CreatedAt:        now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestListContacts{UserId: 100},
			},
			want: &profilev1.ResponseListContacts{
				Contacts: []*profilev1.ContactItem{
					{
						UserId:           100,
						ContactUserId:    101,
						FirstName:        "John",
						LastName:         strPtr("Doe"),
						ContactAvatarUrl: strPtr("avatar.jpg"),
						CreatedAt:        timestamppb.New(now),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.ListContacts(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Len(t, resp.GetContacts(), 1)
		})
	}
}

func TestPositiveProfileServer_AddContact(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestAddContact
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseAddContact
		name    string
		args    args
	}{
		{
			name: "Successful add contact",
			prepare: func(f *fields) {
				f.contactService.EXPECT().AddContact(gomock.Any(), dto.AddContactRequest{
					ContactUserID: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
				}, int64(100)).Return(&dto.ContactResponse{
					UserID:        100,
					ContactUserID: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
					CreatedAt:     now,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
				},
			},
			want: &profilev1.ResponseAddContact{
				Contact: &profilev1.ContactItem{
					UserId:        100,
					ContactUserId: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
					CreatedAt:     timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.AddContact(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestPositiveProfileServer_DeleteContact(t *testing.T) {
	type fields struct {
		profileUseCase *mock.MockProfileServiceInterface
		contactService *mock.MockContactServiceInterface
	}

	type args struct {
		ctx context.Context
		req *profilev1.RequestDeleteContact
	}

	tests := []struct {
		prepare func(*fields)
		want    *profilev1.ResponseDeleteContact
		name    string
		args    args
	}{
		{
			name: "Successful delete contact",
			prepare: func(f *fields) {
				f.contactService.EXPECT().DeleteContact(gomock.Any(), dto.DeleteContactRequest{
					ContactUserID: 101,
				}, int64(100)).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &profilev1.RequestDeleteContact{
					UserId:        100,
					ContactUserId: 101,
				},
			},
			want: &profilev1.ResponseDeleteContact{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileUseCase: mock.NewMockProfileServiceInterface(ctrl),
				contactService: mock.NewMockContactServiceInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileServer{
				profileUseCase: f.profileUseCase,
				contactService: f.contactService,
				logger:         zap.NewNop(),
			}

			resp, err := s.DeleteContact(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestMapContactError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "Cannot create contact with yourself",
			err:      contactdomain.ErrCantCreateContactWithYourself,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "Contact already exists",
			err:      contactdomain.ErrContactExists,
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "Contact not found",
			err:      contactdomain.ErrContactNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "Profile not found",
			err:      pdomain.ErrNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "Unknown error",
			err:      errors.New("unknown"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapContactError(tt.err)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPtrStringOrEmpty(t *testing.T) {
	tests := []struct {
		name string
		v    *string
		want string
	}{
		{
			name: "Nil pointer returns empty",
			v:    nil,
			want: "",
		},
		{
			name: "Non-nil pointer returns value",
			v:    strPtr("test"),
			want: "test",
		},
		{
			name: "Empty string pointer returns empty",
			v:    strPtr(""),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ptrStringOrEmpty(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestContactResponseToProto(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		c    *dto.ContactResponse
		want *profilev1.ContactItem
	}{
		{
			name: "Nil contact returns nil",
			c:    nil,
			want: nil,
		},
		{
			name: "Contact with all fields",
			c: &dto.ContactResponse{
				UserID:           100,
				ContactUserID:    101,
				FirstName:        "John",
				LastName:         strPtr("Doe"),
				ContactAvatarUrl: strPtr("avatar.jpg"),
				CreatedAt:        now,
			},
			want: &profilev1.ContactItem{
				UserId:           100,
				ContactUserId:    101,
				FirstName:        "John",
				LastName:         strPtr("Doe"),
				ContactAvatarUrl: strPtr("avatar.jpg"),
				CreatedAt:        timestamppb.New(now),
			},
		},
		{
			name: "Contact without optional fields",
			c: &dto.ContactResponse{
				UserID:        100,
				ContactUserID: 101,
				FirstName:     "Jane",
				CreatedAt:     now,
			},
			want: &profilev1.ContactItem{
				UserId:        100,
				ContactUserId: 101,
				FirstName:     "Jane",
				CreatedAt:     timestamppb.New(now),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contactResponseToProto(tt.c)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want.GetUserId(), got.GetUserId())
				require.Equal(t, tt.want.GetContactUserId(), got.GetContactUserId())
				require.Equal(t, tt.want.GetFirstName(), got.GetFirstName())
			}
		})
	}
}
