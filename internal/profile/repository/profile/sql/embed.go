package usersql

import _ "embed"

//go:embed create_profile.sql
var CreateProfile string

//go:embed upload_birth_date.sql
var UploadBirthDate string

//go:embed get_profile_id_by_login.sql
var GetProfileIDByLogin string

//go:embed get_profile_by_id.sql
var GetProfileByID string

//go:embed upload_bio.sql
var UploadBio string

//go:embed upload_avatar_url.sql
var UploadAvatarURL string

//go:embed upload_name_first_only.sql
var UploadNameFirstOnly string

//go:embed upload_name_full.sql
var UploadNameFull string

//go:embed delete_user_avatar.sql
var DeleteUserAvatar string

//go:embed update_last_seen.sql
var UpdateLastSeen string
