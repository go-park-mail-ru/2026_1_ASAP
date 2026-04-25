package usersql

import _ "embed"

const CreateUserTxDescription = "tx: EXISTS login; EXISTS email; INSERT users; COMMIT"
const CreateUserByVKIDTxDescription = "tx: EXISTS login; EXISTS email; INSERT users; INSERT vk_accounts; COMMIT"

//go:embed upload_birth_date.sql
var UploadBirthDate string

//go:embed exists_login.sql
var ExistsLogin string

//go:embed exists_email.sql
var ExistsEmail string

//go:embed insert_user.sql
var InsertUser string

//go:embed get_user_by_email.sql
var GetUserByEmail string

//go:embed get_user_by_login.sql
var GetUserByLogin string

//go:embed get_user_by_id.sql
var GetUserByID string

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

//go:embed insert_vk_account.sql
var InsertVKAccount string

//go:embed get_user_by_vk_id.sql
var GetUserByVKID string
