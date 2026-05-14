package messagessql

import _ "embed"

// CreateMessageTxDescription — подпись для лога транзакции CreateMessage (не один SQL).
const CreateMessageTxDescription = "tx: INSERT messages; UPDATE chats last_message; COMMIT"

//go:embed insert_message.sql
var InsertMessage string

//go:embed update_chat_last_message.sql
var UpdateChatLastMessage string

//go:embed get_messages_by_chat_before_id.sql
var GetMessagesByChatBeforeID string

//go:embed get_messages_by_chat.sql
var GetMessagesByChat string
