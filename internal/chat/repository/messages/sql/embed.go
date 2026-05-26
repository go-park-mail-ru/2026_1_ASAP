package messagessql

import _ "embed"

const CreateMessageTxDescription = "tx: INSERT messages; UPDATE chats last_message; COMMIT"

//go:embed insert_message.sql
var InsertMessage string

//go:embed update_message.sql
var UpdateMessage string

//go:embed delete_message.sql
var DeleteMessage string

//go:embed update_chat_last_message.sql
var UpdateChatLastMessage string

//go:embed get_messages_by_chat_before_id.sql
var GetMessagesByChatBeforeID string

//go:embed get_messages_by_chat.sql
var GetMessagesByChat string

//go:embed insert_message_attachment.sql
var InsertMessageAttachment string

//go:embed get_attachments_by_message_ids.sql
var GetAttachmentsByMessageIDs string

//go:embed can_access_attachment.sql
var CanAccessAttachment string

//go:embed update_message_attachment_transcript.sql
var UpdateMessageAttachmentTranscript string

//go:embed get_message_by_id.sql
var GetMessageByID string
