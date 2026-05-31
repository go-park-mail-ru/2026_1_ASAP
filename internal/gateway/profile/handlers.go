package profile

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/jsonbody"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
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

func sendProfileError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch profilev1.ProfileErrorCode(appCode) {
	case profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_TOO_LARGE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_INVALID_TYPE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_EMPTY:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_FIRST_NAME_EMPTY:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_BIRTH_DATE_INVALID:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_BIRTH_DATE_FORMAT:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidDateFormat, Message: dtoApi.InvalidDateFormatMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_SELF:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.ContactWithYourself, Message: dtoApi.ContactWithYourselfMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_ALREADY_EXISTS:
		response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.ContactAlreadyExists, Message: dtoApi.ContactAlreadyExistsMsg}},
		})
	case profilev1.ProfileErrorCode_PROFILE_ERROR_CONTACT_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.ContactNotFound, Message: dtoApi.ContactNotFoundMsg}},
		})
	default:
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func sendAuthError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	if authv1.AuthErrorCode(appCode) == authv1.AuthErrorCode_AUTH_ERROR_USER_NOT_FOUND {
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	} else {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func profileResponseToDTO(resp *profilev1.ResponseGetProfile) *dto.ResponseGetProfile {
	if resp == nil {
		return nil
	}
	out := &dto.ResponseGetProfile{
		UserId:    resp.GetUserId(),
		FirstName: resp.GetFirstName(),
		IsOnline:  resp.GetIsOnline(),
	}
	if resp.LastName != nil {
		out.LastName = resp.LastName
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
	return out
}

func updateProfileResponseToDTO(resp *profilev1.ResponseGetProfile) *dto.ResponseUpdateProfile {
	if resp == nil {
		return nil
	}
	out := &dto.ResponseUpdateProfile{
		UserId:    resp.GetUserId(),
		FirstName: resp.GetFirstName(),
	}
	if resp.LastName != nil {
		out.LastName = resp.LastName
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
	return out
}

func (h *GatewayProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	resp, err := h.ProfileService.GetProfile(ctx, &profilev1.RequestGetProfile{UserId: uid})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	body := profileResponseToDTO(resp)
	if body != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: body.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
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
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	idStr := chi.URLParam(r, "id")
	profileID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidID, Message: dtoApi.InvalidIDMsg}},
		})
		return
	}
	resp, err := h.ProfileService.GetProfile(ctx, &profilev1.RequestGetProfile{UserId: profileID})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	body := profileResponseToDTO(resp)
	if body != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: body.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
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
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	var body dto.RequestUpdateBio
	if err := jsonbody.Decode(r.Body, &body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}
	bio := strings.TrimSpace(*body.Bio)
	resp, err := h.ProfileService.UpdateProfileBio(ctx, &profilev1.RequestUpdateBio{
		UserId: uid,
		Bio:    &bio,
	})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	out := updateProfileResponseToDTO(resp)
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
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
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileNotFound, Message: dtoApi.FileNotFoundMsg}},
		})
		return
	}
	fileInput, err := media.FileInputFromMultipart(file, header)
	if err != nil {
		if errors.Is(err, media.ErrEmptyFile) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}},
			})
			return
		}
		if errors.Is(err, media.ErrFileTooLarge) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
			})
			return
		}
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}
	content, err := io.ReadAll(fileInput.Body)
	if err != nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}
	resp, err := h.ProfileService.UpdateProfileAvatar(ctx, &profilev1.RequestUpdateAvatar{
		UserId:  uid,
		Content: content,
		Type:    fileInput.ContentType,
	})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	out := updateProfileResponseToDTO(resp)
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
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
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	var body dto.RequestUpdateBirthDate
	if err := jsonbody.Decode(r.Body, &body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}
	if body.BirthDate == nil || strings.TrimSpace(*body.BirthDate) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidDate, Message: dtoApi.InvalidDateMsg}},
		})
		return
	}
	bd := strings.TrimSpace(*body.BirthDate)
	resp, err := h.ProfileService.UpdateProfileBirthDate(ctx, &profilev1.RequestUpdateBirthDate{
		UserId:    uid,
		BirthDate: &bd,
	})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	out := updateProfileResponseToDTO(resp)
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
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
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	var body dto.RequestUpdateName
	if err := jsonbody.Decode(r.Body, &body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}
	if strings.TrimSpace(body.FirstName) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFirstName, Message: dtoApi.EmptyFirstNameMsg}},
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
		sendProfileError(w, err)
		return
	}
	out := updateProfileResponseToDTO(resp)
	if out != nil {
		pub, errAuth := h.AuthService.GetUserPublic(ctx, &authv1.RequestGetUserPublic{UserId: out.UserId})
		if errAuth != nil {
			sendAuthError(w, errAuth)
			return
		}
		out.Login = pub.GetLogin()
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   out,
	})
}

func (h *GatewayProfileHandler) SearchIdByLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	login := strings.TrimSpace(r.URL.Query().Get("login"))
	if login == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyLogin, Message: dtoApi.EmptyLoginMsg}},
		})
		return
	}
	resp, err := h.ProfileService.SearchIdByLogin(ctx, &profilev1.RequestSearchIdByLogin{Login: login})
	if err != nil {
		sendProfileError(w, err)
		return
	}
	var body *dto.ResponseSearchIdByLogin
	if resp != nil {
		body = &dto.ResponseSearchIdByLogin{
			UserId: resp.GetUserId(),
			Login:  resp.GetLogin(),
		}
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseSearchIdByLogin]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayProfileHandler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value(middleware.UserID).(int64); !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}
	response.Send(w, http.StatusNotImplemented, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.NotImplemented, Message: dtoApi.NotImplementedMsg}},
	})
}
