package vkid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestAuthFromPublicInfoJSON_nested(t *testing.T) {
	const docExample = `{
    "profile": {
        "user_id": "123456789",
        "first_name": "Иван",
        "last_name": "И",
        "phone": "+7 *** *** ** 42",
        "avatar": "https://example.com/photo.jpg",
        "email": "masked@***.ru"
    }
}`
	got, err := RequestAuthFromPublicInfoJSON([]byte(docExample), 0)
	require.NoError(t, err)
	require.Equal(t, int64(123456789), got.VKUserID)
	require.Equal(t, "Иван", got.FirstName)
	require.Equal(t, "И", got.LastName)
	require.Equal(t, "masked@***.ru", got.Email)
	require.Equal(t, "https://example.com/photo.jpg", got.AvatarURL)
}

func TestRequestAuthFromPublicInfoJSON_nestedNumericUserID(t *testing.T) {
	raw := `{"profile":{"user_id":987654321,"first_name":"Ann"}}`
	got, err := RequestAuthFromPublicInfoJSON([]byte(raw), 0)
	require.NoError(t, err)
	require.Equal(t, int64(987654321), got.VKUserID)
	require.Equal(t, "Ann", got.FirstName)
}

func TestRequestAuthFromPublicInfoJSON_flatFallback(t *testing.T) {
	raw := `{"user_id":555,"email":"a@b.c","first_name":"Bob","last_name":"Q","avatar":"http://x"}`
	got, err := RequestAuthFromPublicInfoJSON([]byte(raw), 0)
	require.NoError(t, err)
	require.Equal(t, int64(555), got.VKUserID)
	require.Equal(t, "a@b.c", got.Email)
}

func TestRequestAuthFromPublicInfoJSON_nestedOAuthFallbackPhoneOnly(t *testing.T) {
	raw := `{"profile":{"phone":"+7 *** ***","avatar":"http://photo"}}`
	got, err := RequestAuthFromPublicInfoJSON([]byte(raw), 424242424)
	require.NoError(t, err)
	require.Equal(t, int64(424242424), got.VKUserID)
	require.Equal(t, "http://photo", got.AvatarURL)
}

func TestRequestAuthFromPublicInfoJSON_apiError(t *testing.T) {
	raw := `{"error":"invalid_id_token","error_description":"bad token"}`
	_, err := RequestAuthFromPublicInfoJSON([]byte(raw), 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_id_token")
}

func TestRequestAuthFromPublicInfoJSON_userWithAltFieldNames(t *testing.T) {
	raw := `{"user":{"id":"42","given_name":"John","family_name":"Doe","mail":"john@doe.test","avatar_url":"https://example/avatar.png"}}`
	got, err := RequestAuthFromPublicInfoJSON([]byte(raw), 0)
	require.NoError(t, err)
	require.Equal(t, int64(42), got.VKUserID)
	require.Equal(t, "John", got.FirstName)
	require.Equal(t, "Doe", got.LastName)
	require.Equal(t, "john@doe.test", got.Email)
	require.Equal(t, "https://example/avatar.png", got.AvatarURL)
}
