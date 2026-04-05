package profile

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	media "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
	mock "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/profile/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestPositiveProfileService_GetUserProfile(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *dto.ResponseGetProfile
	}{
		{
			name: "Base profile information",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return(&domain.Profile{UserId: 123, Login: "danil_kolbasenko", Avatar: nil, Bio: nil, LastSeen: nil}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:   123,
				Login:    "danil_kolbasenko",
				Avatar:   nil,
				Bio:      nil,
				LastSeen: nil,
			},
		},
		{
			name: "Profile information with Bio",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return(&domain.Profile{UserId: 123, Login: "danil_kolbasenko", Avatar: nil, Bio: strPtr("profile bio"), LastSeen: nil}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:   123,
				Login:    "danil_kolbasenko",
				Avatar:   nil,
				Bio:      strPtr("profile bio"),
				LastSeen: nil,
			},
		},
		{
			name: "Profile information with Bio, avatar",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return(&domain.Profile{UserId: 123, Login: "danil_kolbasenko", Avatar: strPtr("s3 avatar url"), Bio: strPtr("profile bio"), LastSeen: nil}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:   123,
				Login:    "danil_kolbasenko",
				Avatar:   strPtr("s3 avatar url"),
				Bio:      strPtr("profile bio"),
				LastSeen: nil,
			},
		},
		{
			name: "Profile information full",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return(&domain.Profile{UserId: 123, Login: "danil_kolbasenko", Avatar: strPtr("s3 avatar url"), Bio: strPtr("profile bio"), BirthDate: timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)), LastSeen: timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC))}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:    123,
				Login:     "danil_kolbasenko",
				Avatar:    strPtr("s3 avatar url"),
				Bio:       strPtr("profile bio"),
				LastSeen:  timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)),
				BirthDate: strPtr("2026-04-01"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.GetUserProfile(tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeProfileService_GetUserProfile(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "User doesn't exist",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return((*domain.Profile)(nil), domain.ErrNotFound)
			},
			args:    args{ctx: context.Background(), userID: 123},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).Return((*domain.Profile)(nil), errors.New("unknown error"))
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
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.GetUserProfile(tt.args.ctx, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestPositiveProfileService_UpdateBio(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateBio
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *dto.ResponseUpdateProfile
	}{
		{
			name: "Usual bio",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().UploadBio(context.Background(), int64(123), "sample bio").Return(&domain.Profile{UserId: 123, Login: "danil_kolbasenko", Bio: strPtr("sample bio")}, nil)
			},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBio{strPtr("sample bio")}},
			want: &dto.ResponseUpdateProfile{
				UserId: 123,
				Login:  "danil_kolbasenko",
				Bio:    strPtr("sample bio"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.UpdateProfileBio(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeProfileService_UpdateBio(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateBio
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "Empty bio",
			prepare: func(f *fields) {
			},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBio{Bio: nil}},
			wantErr: domain.ErrEmptyBio,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().UploadBio(context.Background(), int64(123), "test bio").Return((*domain.Profile)(nil), errors.New("unknown error"))
			},
			args:       args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBio{Bio: strPtr("test bio")}},
			wantAnyErr: true,
		},
		{
			name:       "Nil request",
			prepare:    func(f *fields) {},
			args:       args{ctx: context.Background(), userID: 123, request: nil},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.UpdateProfileBio(tt.args.ctx, tt.args.userID, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestPositiveProfileService_UpdateAvatar(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateAvatar
	}

	fileInput := &media.FileInput{
		Body:        bytes.NewBufferString("avatar-bytes"),
		ContentType: "image/png",
		Size:        1024,
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *dto.ResponseUpdateProfile
	}{
		{
			name: "Usual avatar",
			prepare: func(f *fields) {
				f.mediaRepository.EXPECT().
					UploadAvatar(context.Background(), int64(123), fileInput).
					Return("avatar-url", nil)

				f.profileRepository.EXPECT().
					UploadAvatarUrl(context.Background(), int64(123), "avatar-url").
					Return(&domain.Profile{
						UserId: 123,
						Login:  "danil_kolbasenko",
						Avatar: strPtr("avatar-url"),
						Bio:    strPtr("sample bio"),
					}, nil)
			},
			args: func() args {
				return args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: fileInput}}
			}(),
			want: &dto.ResponseUpdateProfile{
				UserId: 123,
				Login:  "danil_kolbasenko",
				Avatar: strPtr("avatar-url"),
				Bio:    strPtr("sample bio"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.UpdateProfileAvatar(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeProfileService_UpdateAvatar(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateAvatar
	}

	fileInputUploadErr := &media.FileInput{
		Body:        bytes.NewBufferString("avatar-bytes"),
		ContentType: "image/png",
		Size:        1024,
	}
	fileInputUploadUrlErr := &media.FileInput{
		Body:        bytes.NewBufferString("avatar-bytes"),
		ContentType: "image/png",
		Size:        1024,
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:       "Nil request",
			prepare:    func(f *fields) {},
			args:       args{ctx: context.Background(), userID: 123, request: nil},
			wantAnyErr: true,
		},
		{
			name:    "Empty file - nil file input",
			prepare: func(f *fields) {},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: nil}},
			wantErr: domain.ErrEmptyAvatar,
		},
		{
			name:    "Empty file - nil body",
			prepare: func(f *fields) {},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: &media.FileInput{
				Body:        nil,
				ContentType: "image/png",
				Size:        1024,
			}}},
			wantErr: domain.ErrEmptyAvatar,
		},
		{
			name:    "Empty file - zero size",
			prepare: func(f *fields) {},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: &media.FileInput{
				Body:        bytes.NewBufferString("avatar-bytes"),
				ContentType: "image/png",
				Size:        0,
			}}},
			wantErr: domain.ErrEmptyAvatar,
		},
		{
			name:    "Invalid file type",
			prepare: func(f *fields) {},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: &media.FileInput{
				Body:        bytes.NewBufferString("avatar-bytes"),
				ContentType: "application/octet-stream",
				Size:        1024,
			}}},
			wantErr: domain.ErrInvalidAvatarType,
		},
		{
			name:    "Avatar too large",
			prepare: func(f *fields) {},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: &media.FileInput{
				Body:        bytes.NewBufferString("avatar-bytes"),
				ContentType: "image/png",
				Size:        5*1024*1024 + 1,
			}}},
			wantErr: domain.ErrAvatarTooLarge,
		},
		{
			name: "Upload avatar error",
			prepare: func(f *fields) {
				f.mediaRepository.EXPECT().
					UploadAvatar(context.Background(), int64(123), fileInputUploadErr).
					Return("", errors.New("upload error"))
			},
			args:       args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: fileInputUploadErr}},
			wantAnyErr: true,
		},
		{
			name: "Upload avatar url error",
			prepare: func(f *fields) {
				f.mediaRepository.EXPECT().
					UploadAvatar(context.Background(), int64(123), fileInputUploadUrlErr).
					Return("avatar-url", nil)
				f.profileRepository.EXPECT().
					UploadAvatarUrl(context.Background(), int64(123), "avatar-url").
					Return((*domain.Profile)(nil), errors.New("db error"))
			},
			args:       args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: fileInputUploadUrlErr}},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}

			result, err := s.UpdateProfileAvatar(tt.args.ctx, tt.args.userID, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestPositiveProfileService_UpdateBirthDate(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateBirthDate
	}

	wantBirthParsed, err := time.Parse(time.DateOnly, "1995-03-10")
	require.NoError(t, err)

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *dto.ResponseUpdateProfile
	}{
		{
			name: "Sets birth date",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					UploadBirthDate(context.Background(), int64(123), &wantBirthParsed).
					Return(&domain.Profile{
						UserId:    123,
						Login:     "danil_kolbasenko",
						FirstName: "Ivan",
						LastName:  strPtr("Petrov"),
						BirthDate: &wantBirthParsed,
						Avatar:    strPtr("av"),
						Bio:       strPtr("bio text"),
						LastSeen:  timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)),
					}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  123,
				request: &dto.RequestUpdateBirthDate{BirthDate: strPtr("1995-03-10")},
			},
			want: &dto.ResponseUpdateProfile{
				UserId:    123,
				Login:     "danil_kolbasenko",
				FirstName: "Ivan",
				LastName:  strPtr("Petrov"),
				BirthDate: strPtr("1995-03-10"),
				Avatar:    strPtr("av"),
				Bio:       strPtr("bio text"),
				LastSeen:  timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "Repository returns profile without birth date pointer",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					UploadBirthDate(context.Background(), int64(123), &wantBirthParsed).
					Return(&domain.Profile{
						UserId:    123,
						Login:     "u",
						FirstName: "A",
						BirthDate: nil,
					}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  123,
				request: &dto.RequestUpdateBirthDate{BirthDate: strPtr("1995-03-10")},
			},
			want: &dto.ResponseUpdateProfile{
				UserId:    123,
				Login:     "u",
				FirstName: "A",
				BirthDate: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			result, err := s.UpdateProfileBirthDate(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeProfileService_UpdateBirthDate(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateBirthDate
	}

	futureDate := time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)

	repoBirth2000, err := time.Parse(time.DateOnly, "2000-01-01")
	require.NoError(t, err)

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:       "Nil request",
			prepare:    func(f *fields) {},
			args:       args{ctx: context.Background(), userID: 123, request: nil},
			wantAnyErr: true,
		},
		{
			name:    "Nil birth date in request",
			prepare: func(f *fields) {},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBirthDate{BirthDate: nil}},
			wantErr: domain.ErrInvalidBirthDate,
		},
		{
			name:    "Invalid date format",
			prepare: func(f *fields) {},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBirthDate{BirthDate: strPtr("31-13-2000")}},
			wantErr: domain.ErrInvalidBirthDateFormat,
		},
		{
			name:    "Birth date in the future",
			prepare: func(f *fields) {},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBirthDate{BirthDate: strPtr(futureDate)}},
			wantErr: domain.ErrInvalidBirthDate,
		},
		{
			name: "Repository returns invalid birth date",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					UploadBirthDate(context.Background(), int64(123), &repoBirth2000).
					Return((*domain.Profile)(nil), domain.ErrInvalidBirthDate)
			},
			args:    args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBirthDate{BirthDate: strPtr("2000-01-01")}},
			wantErr: domain.ErrInvalidBirthDate,
		},
		{
			name: "Repository unknown error",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					UploadBirthDate(context.Background(), int64(123), &repoBirth2000).
					Return((*domain.Profile)(nil), errors.New("db down"))
			},
			args:       args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBirthDate{BirthDate: strPtr("2000-01-01")}},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}

			result, err := s.UpdateProfileBirthDate(tt.args.ctx, tt.args.userID, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestPositiveProfileService_SearchIdByLogin(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestSearchIdByLogin
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *dto.ResponseSearchIdByLogin
	}{
		{
			name: "found by login",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					GetProfileIdByLogin(context.Background(), "alice").
					Return(int64(42), nil)
			},
			args: args{
				ctx:     context.Background(),
				request: &dto.RequestSearchIdByLogin{Login: "alice"},
			},
			want: &dto.ResponseSearchIdByLogin{UserId: 42, Login: "alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			got, err := s.SearchIdByLogin(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeProfileService_SearchIdByLogin(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestSearchIdByLogin
	}

	tests := []struct {
		name           string
		prepare        func(*fields)
		args           args
		wantErrIs      error
		wantErrMessage string
		wantAnyErr     bool
	}{
		{
			name:    "nil request",
			prepare: func(f *fields) {},
			args: args{
				ctx:     context.Background(),
				request: nil,
			},
			wantErrMessage: "update profile bio nil request",
		},
		{
			name: "user not found",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					GetProfileIdByLogin(context.Background(), "nobody").
					Return(int64(0), domain.ErrNotFound)
			},
			args: args{
				ctx:     context.Background(),
				request: &dto.RequestSearchIdByLogin{Login: "nobody"},
			},
			wantErrIs: domain.ErrNotFound,
		},
		{
			name: "repository error",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					GetProfileIdByLogin(context.Background(), "alice").
					Return(int64(0), errors.New("db down"))
			},
			args: args{
				ctx:     context.Background(),
				request: &dto.RequestSearchIdByLogin{Login: "alice"},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaRepositoryInterface(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			got, err := s.SearchIdByLogin(tt.args.ctx, tt.args.request)
			require.Nil(t, got)
			if tt.wantAnyErr {
				require.Error(t, err)
				return
			}
			if tt.wantErrIs != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tt.wantErrIs), "expected %v, got %v", tt.wantErrIs, err)
				return
			}
			require.EqualError(t, err, tt.wantErrMessage)
		})
	}
}
