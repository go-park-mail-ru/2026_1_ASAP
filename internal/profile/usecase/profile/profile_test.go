package profile

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	media "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/usecase/profile/mock"
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseGetProfile
		args    args
		name    string
	}{
		{
			name: "Base profile information",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).
					Return(&domain.Profile{UserId: 123, Avatar: nil, Bio: nil, LastSeen: nil, FirstName: "danil"}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:    123,
				FirstName: "danil",
				Avatar:    nil,
				Bio:       nil,
				LastSeen:  nil,
			},
		},
		{
			name: "Profile information with Bio",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).
					Return(&domain.Profile{UserId: 123, Avatar: nil, Bio: strPtr("profile bio"), LastSeen: nil, FirstName: "danil"}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:    123,
				FirstName: "danil",
				Avatar:    nil,
				Bio:       strPtr("profile bio"),
				LastSeen:  nil,
			},
		},
		{
			name: "Profile information with Bio, avatar",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).
					Return(&domain.Profile{UserId: 123, Avatar: strPtr("s3 avatar url"), Bio: strPtr("profile bio"), LastSeen: nil, FirstName: "danil"}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:    123,
				FirstName: "danil",
				Avatar:    strPtr("s3 avatar url"),
				Bio:       strPtr("profile bio"),
				LastSeen:  nil,
			},
		},
		{
			name: "Profile information full",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().GetProfileById(context.Background(), int64(123)).
					Return(&domain.Profile{
						UserId:    123,
						Avatar:    strPtr("s3 avatar url"),
						Bio:       strPtr("profile bio"),
						BirthDate: timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)),
						LastSeen:  timePtr(time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)),
						FirstName: "danil",
					}, nil)
			},
			args: args{ctx: context.Background(), userID: 123},
			want: &dto.ResponseGetProfile{
				UserId:    123,
				FirstName: "danil",
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		args       args
		name       string
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateBio
		userID  int64
	}

	tests := []struct {
		args    args
		prepare func(*fields)
		want    *dto.ResponseUpdateProfile
		name    string
	}{
		{
			name: "Usual bio",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().UploadBio(context.Background(), int64(123), "sample bio").
					Return(&domain.Profile{UserId: 123, Bio: strPtr("sample bio"), FirstName: "danil"}, nil)
			},
			args: args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateBio{Bio: strPtr("sample bio")}},
			want: &dto.ResponseUpdateProfile{
				UserId:    123,
				FirstName: "danil",
				Bio:       strPtr("sample bio"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateBio
		userID  int64
	}

	tests := []struct {
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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

func TestPositiveProfileService_UpdateBioEscapesHTML(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaService
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
		wantBio string
	}{
		{
			name: "script and quotes",
			prepare: func(f *fields) {
				raw := `<script>alert("xss")</script>`
				f.profileRepository.EXPECT().
					UploadBio(context.Background(), int64(123), raw).
					Return(&domain.Profile{
						UserId: 123,
						Bio:    strPtr(raw),
					}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  123,
				request: &dto.RequestUpdateBio{Bio: strPtr(`<script>alert("xss")</script>`)},
			},
			wantBio: `<script>alert("xss")</script>`,
		},
		{
			name: "ampersand",
			prepare: func(f *fields) {
				raw := `Tom & Jerry`
				f.profileRepository.EXPECT().
					UploadBio(context.Background(), int64(123), raw).
					Return(&domain.Profile{
						UserId: 123,
						Bio:    strPtr(raw),
					}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  123,
				request: &dto.RequestUpdateBio{Bio: strPtr(`Tom & Jerry`)},
			},
			wantBio: `Tom & Jerry`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			resp, err := s.UpdateProfileBio(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.wantBio, *resp.Bio)
		})
	}
}

func TestPositiveProfileService_UpdateNameEscapesHTML(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request *dto.RequestUpdateName
	}

	tests := []struct {
		name        string
		prepare     func(*fields)
		args        args
		wantFirst   string
		wantLast    string
		wantNilLast bool
	}{
		{
			name: "markup in first and last name",
			prepare: func(f *fields) {
				first := `<b>Ann</b>`
				last := `<img src=x onerror=alert(1)>`
				lastPtr := &last
				f.profileRepository.EXPECT().
					UploadName(context.Background(), int64(321), first, lastPtr).
					Return(&domain.Profile{
						UserId:    321,
						FirstName: first,
						LastName:  lastPtr,
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 321,
				request: &dto.RequestUpdateName{
					FirstName: `<b>Ann</b>`,
					LastName:  strPtr(`<img src=x onerror=alert(1)>`),
				},
			},
			wantFirst:   `<b>Ann</b>`,
			wantLast:    `<img src=x onerror=alert(1)>`,
			wantNilLast: false,
		},
		{
			name: "ampersand in first name only",
			prepare: func(f *fields) {
				first := `A & B`
				f.profileRepository.EXPECT().
					UploadName(context.Background(), int64(500), first, nil).
					Return(&domain.Profile{
						UserId:    500,
						FirstName: first,
						LastName:  nil,
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 500,
				request: &dto.RequestUpdateName{
					FirstName: `A & B`,
					LastName:  nil,
				},
			},
			wantFirst:   `A & B`,
			wantNilLast: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			resp, err := s.UpdateProfileName(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.wantFirst, resp.FirstName)
			if tt.wantNilLast {
				require.Nil(t, resp.LastName)
			} else {
				require.Equal(t, tt.wantLast, *resp.LastName)
			}
		})
	}
}

func TestPositiveProfileService_GetUserProfileEscapesLogin(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name      string
		prepare   func(*fields)
		args      args
		wantLogin string
	}{
		{
			name: "angle brackets",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					GetProfileById(context.Background(), int64(7)).
					Return(&domain.Profile{
						UserId:    7,
						FirstName: "f",
					}, nil)
			},
			args: args{ctx: context.Background(), userID: 7},
		},
		{
			name: "ampersand",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					GetProfileById(context.Background(), int64(8)).
					Return(&domain.Profile{
						UserId:    8,
						FirstName: "x",
					}, nil)
			},
			args: args{ctx: context.Background(), userID: 8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileRepository: mock.NewMockProfileRepositoryInterface(ctrl),
				mediaRepository:   mock.NewMockMediaService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ProfileService{
				profileRepository: f.profileRepository,
				mediaRepository:   f.mediaRepository,
			}
			resp, err := s.GetUserProfile(tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.wantLogin, resp.Login)
		})
	}
}

func TestPositiveProfileService_UpdateAvatar(t *testing.T) {
	type fields struct {
		profileRepository *mock.MockProfileRepositoryInterface
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateAvatar
		userID  int64
	}

	fileInput := &media.FileInput{
		Body:        bytes.NewBufferString("avatar-bytes"),
		ContentType: "image/png",
		Size:        1024,
	}

	tests := []struct {
		args    args
		prepare func(*fields)
		want    *dto.ResponseUpdateProfile
		name    string
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
						Avatar: strPtr("avatar-url"),
						Bio:    strPtr("sample bio"),
					}, nil)
			},
			args: func() args {
				return args{ctx: context.Background(), userID: 123, request: &dto.RequestUpdateAvatar{File: fileInput}}
			}(),
			want: &dto.ResponseUpdateProfile{
				UserId: 123,
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateAvatar
		userID  int64
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
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateBirthDate
		userID  int64
	}

	wantBirthParsed, err := time.Parse(time.DateOnly, "1995-03-10")
	require.NoError(t, err)

	tests := []struct {
		args    args
		prepare func(*fields)
		want    *dto.ResponseUpdateProfile
		name    string
	}{
		{
			name: "Sets birth date",
			prepare: func(f *fields) {
				f.profileRepository.EXPECT().
					UploadBirthDate(context.Background(), int64(123), &wantBirthParsed).
					Return(&domain.Profile{
						UserId:    123,
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateBirthDate
		userID  int64
	}

	futureDate := time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)

	repoBirth2000, err := time.Parse(time.DateOnly, "2000-01-01")
	require.NoError(t, err)

	tests := []struct {
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestSearchIdByLogin
	}

	tests := []struct {
		args    args
		prepare func(*fields)
		want    *dto.ResponseSearchIdByLogin
		name    string
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
		mediaRepository   *mock.MockMediaService
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
			wantErrMessage: "search id by login: nil request",
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
				mediaRepository:   mock.NewMockMediaService(ctrl),
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
