package chatssql

import _ "embed"

//go:embed get_all_chats_by_user_id.sql
var GetAllChatsByUserID string

//go:embed get_chat_by_id.sql
var GetChatByID string

//go:embed insert_chat.sql
var InsertChat string

//go:embed get_last_message_of_chat.sql
var GetLastMessageOfChat string

//go:embed get_last_messages_of_chats.sql
var GetLastMessagesOfChats string

//go:embed get_chat_members.sql
var GetChatMembers string

//go:embed insert_chat_member.sql
var InsertChatMember string

//go:embed is_member.sql
var IsMember string

//go:embed get_dialog_between_users.sql
var GetDialogBetweenUsers string

//go:embed delete_messages_by_chat_id.sql
var DeleteMessagesByChatID string

//go:embed delete_chat_members_by_chat_id.sql
var DeleteChatMembersByChatID string

//go:embed delete_chat_by_id.sql
var DeleteChatByID string

//go:embed get_member_role.sql
var GetMemberRole string

//go:embed update_chat_avatar_url.sql
var UpdateChatAvatarURL string

//go:embed update_chat_title.sql
var UpdateChatTitle string

//go:embed update_chat_description.sql
var UpdateChatDescription string

//go:embed delete_chat_member.sql
var DeleteChatMember string
