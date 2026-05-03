package searchsql

import _ "embed"

//go:embed search_chats.sql
var SearchChats string

//go:embed search_global_channels.sql
var SearchGlobalChannels string

//go:embed search_contacts.sql
var SearchContacts string

//go:embed search_users.sql
var SearchUsers string

//go:embed search_messages_in_chat.sql
var SearchMessagesInChat string
