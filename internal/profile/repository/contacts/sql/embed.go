package contactssql

import _ "embed"

//go:embed get_all_contacts_by_user_id.sql
var GetAllContactsByUserID string

//go:embed insert_contact.sql
var InsertContact string

//go:embed get_user_avatar_url.sql
var GetUserAvatarURL string

//go:embed delete_contact.sql
var DeleteContact string

//go:embed is_contact.sql
var IsContact string

//go:embed get_contact.sql
var GetContact string
