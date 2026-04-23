package profile

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GatewayProfileHandler struct {
	AuthService authv1.AuthClient

	ProfileService profilev1.ProfileClient
}

func NewGatewayProfileHandler(auth authv1.AuthClient, profile profilev1.ProfileClient) *GatewayProfileHandler {
	return &GatewayProfileHandler{
		AuthService:    auth,
		ProfileService: profile,
	}
}

func (h *GatewayProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	resp, err := h.ProfileService.GetProfile(ctx, &profilev1.RequestGetProfile{UserId: uid})
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var body *dto.ResponseGetProfile
	if resp != nil {
		body = &dto.ResponseGetProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
			Email:     "",
		}
		if resp.SecondName != nil {
			body.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			body.Bio = &b
		}
		if bd := resp.GetBirthDate(); bd != "" {
			body.BirthDate = &bd
		}
		if a := resp.GetAvatar(); a != "" {
			body.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			body.LastSeen = &t
		}
	}
	if body != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: body.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		body.Login = pub.GetLogin()
		body.Email = pub.GetEmail()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseGetProfile]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayProfileHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := ctx.Value(middleware.UserID).(int64); !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	idStr := chi.URLParam(r, "id")
	profileID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidID,
				Message: dtoApi.InvalidIDMsg,
			}},
		})
		return
	}
	resp, err := h.ProfileService.GetProfile(ctx, &profilev1.RequestGetProfile{UserId: profileID})
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var body *dto.ResponseGetProfile
	if resp != nil {
		body = &dto.ResponseGetProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
			Email:     "",
		}
		if resp.SecondName != nil {
			body.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			body.Bio = &b
		}
		if bd := resp.GetBirthDate(); bd != "" {
			body.BirthDate = &bd
		}
		if a := resp.GetAvatar(); a != "" {
			body.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			body.LastSeen = &t
		}
	}
	if body != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: body.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		body.Login = pub.GetLogin()
		body.Email = pub.GetEmail()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseGetProfile]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayProfileHandler) UpdateUserBio(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	var body dto.RequestUpdateBio
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}
	if body.Bio == nil || strings.TrimSpace(*body.Bio) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.EmptyBIO,
				Message: dtoApi.EmptyBIOMsg,
			}},
		})
		return
	}
	bio := strings.TrimSpace(*body.Bio)
	resp, err := h.ProfileService.UpdateProfileBio(ctx, &profilev1.RequestUpdateBio{
		UserId: uid,
		Bio:    &bio,
	})
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var out *dto.ResponseUpdateProfile
	if resp != nil {
		out = &dto.ResponseUpdateProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
		}
		if resp.SecondName != nil {
			out.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			out.Bio = &b
		}
		if bd := resp.GetBirthDate(); bd != "" {
			out.BirthDate = &bd
		}
		if a := resp.GetAvatar(); a != "" {
			out.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			out.LastSeen = &t
		}
	}
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		out.Login = pub.GetLogin()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   out,
	})
}

func (h *GatewayProfileHandler) UpdateUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.FileNotFound,
				Message: dtoApi.FileNotFoundMsg,
			}},
		})
		return
	}
	fileInput, err := media.FileInputFromMultipart(file, header)
	if err != nil {
		if errors.Is(err, media.ErrEmptyFile) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.EmptyFile,
					Message: dtoApi.EmptyFileMsg,
				}},
			})
			return
		}
		if errors.Is(err, media.ErrFileTooLarge) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.FileTooLarge,
					Message: dtoApi.FileTooLargeMsg,
				}},
			})
			return
		}
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InternalError,
				Message: dtoApi.InternalErrorMsg,
			}},
		})
		return
	}
	content, err := io.ReadAll(fileInput.Body)
	if err != nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InternalError,
				Message: dtoApi.InternalErrorMsg,
			}},
		})
		return
	}
	resp, err := h.ProfileService.UpdateProfileAvatar(ctx, &profilev1.RequestUpdateAvatar{
		UserId:  uid,
		Content: content,
		Type:    fileInput.ContentType,
	})
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var out *dto.ResponseUpdateProfile
	if resp != nil {
		out = &dto.ResponseUpdateProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
		}
		if resp.SecondName != nil {
			out.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			out.Bio = &b
		}
		if bd := resp.GetBirthDate(); bd != "" {
			out.BirthDate = &bd
		}
		if a := resp.GetAvatar(); a != "" {
			out.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			out.LastSeen = &t
		}
	}
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		out.Login = pub.GetLogin()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   out,
	})
}

func (h *GatewayProfileHandler) UpdateUserBirthDate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	var body dto.RequestUpdateBirthDate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}
	if body.BirthDate == nil || strings.TrimSpace(*body.BirthDate) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidDate,
				Message: dtoApi.InvalidDateMsg,
			}},
		})
		return
	}
	bd := strings.TrimSpace(*body.BirthDate)
	resp, err := h.ProfileService.UpdateProfileBirthDate(ctx, &profilev1.RequestUpdateBirthDate{
		UserId:    uid,
		BirthDate: &bd,
	})
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var out *dto.ResponseUpdateProfile
	if resp != nil {
		out = &dto.ResponseUpdateProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
		}
		if resp.SecondName != nil {
			out.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			out.Bio = &b
		}
		if bda := resp.GetBirthDate(); bda != "" {
			out.BirthDate = &bda
		}
		if a := resp.GetAvatar(); a != "" {
			out.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			out.LastSeen = &t
		}
	}
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		out.Login = pub.GetLogin()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   out,
	})
}

func (h *GatewayProfileHandler) UpdateProfileName(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	var body dto.RequestUpdateName
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}
	if strings.TrimSpace(body.FirstName) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.EmptyFirstName,
				Message: dtoApi.EmptyFirstNameMsg,
			}},
		})
		return
	}
	req := &profilev1.RequestUpdateName{
		UserId:    uid,
		FirstName: strings.TrimSpace(body.FirstName),
	}
	if body.LastName != nil {
		ln := strings.TrimSpace(*body.LastName)
		req.SecondName = &ln
	}
	resp, err := h.ProfileService.UpdateProfileName(ctx, req)
	if err != nil {
		st, stOK := status.FromError(err)
		if !stOK {
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
			return
		}
		switch st.Code() {
		case codes.NotFound:
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.NotFound,
					Message: dtoApi.NotFoundMsg,
				}},
			})
		case codes.InvalidArgument:
			lower := strings.ToLower(st.Message())
			var errs []dtoApi.ApiError
			switch {
			case strings.Contains(lower, "avatar") || strings.Contains(lower, "content"):
				if strings.Contains(lower, "large") {
					errs = []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}}
				} else if strings.Contains(lower, "type") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}}
				}
			case strings.Contains(lower, "first"):
				errs = []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}}
			case strings.Contains(lower, "birth") || strings.Contains(lower, "date"):
				if strings.Contains(lower, "format") {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}}
				} else {
					errs = []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}}
				}
			default:
				errs = []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}}
			}
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{Status: dtoApi.Error, Errors: errs})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				}},
			})
		}
		return
	}
	var out *dto.ResponseUpdateProfile
	if resp != nil {
		out = &dto.ResponseUpdateProfile{
			UserId:    resp.GetUserId(),
			FirstName: resp.GetFirstName(),
			Login:     "",
		}
		if resp.SecondName != nil {
			out.LastName = resp.SecondName
		}
		if b := resp.GetBio(); b != "" {
			out.Bio = &b
		}
		if bda := resp.GetBirthDate(); bda != "" {
			out.BirthDate = &bda
		}
		if a := resp.GetAvatar(); a != "" {
			out.Avatar = &a
		}
		if ts := resp.GetLastSeen(); ts != nil {
			t := ts.AsTime()
			out.LastSeen = &t
		}
	}
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			st, stOK := status.FromError(errAuth)
			if !stOK {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
				return
			}
			switch st.Code() {
			case codes.NotFound:
				response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
					}},
				})
			default:
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InternalError,
						Message: dtoApi.InternalErrorMsg,
					}},
				})
			}
			return
		}
		out.Login = pub.GetLogin()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   out,
	})
}

// SearchIdByLogin — в proto нет RPC; оставлено под будущую реализацию.
func (h *GatewayProfileHandler) SearchIdByLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.URL.Query().Get("login")
	response.Send(w, http.StatusNotImplemented, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{
			Code:    dtoApi.NotImplemented,
			Message: dtoApi.NotImplementedMsg,
		}},
	})
}

// DeleteUserAvatar — в proto нет RPC; оставлено под будущую реализацию.
func (h *GatewayProfileHandler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value(middleware.UserID).(int64); !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}
	response.Send(w, http.StatusNotImplemented, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{
			Code:    dtoApi.NotImplemented,
			Message: dtoApi.NotImplementedMsg,
		}},
	})
}
