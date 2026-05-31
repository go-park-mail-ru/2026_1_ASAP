package subscriptionsql

import _ "embed"

//go:embed get_by_user_id.sql
var GetByUserID string

//go:embed upsert.sql
var Upsert string
