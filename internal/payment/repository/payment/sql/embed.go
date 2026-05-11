package paymentsql

import _ "embed"

//go:embed insert.sql
var Insert string

//go:embed get_by_payment_id.sql
var GetByPaymentID string

//go:embed get_by_id.sql
var GetByID string

//go:embed update_by_payment_id.sql
var UpdateByPaymentID string
