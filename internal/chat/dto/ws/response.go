package ws

type WsResponseStatus string

const (
	Error       WsResponseStatus = "error"
	MessageNew  WsResponseStatus = "message.New"
	MessageGet  WsResponseStatus = "message.Get"
	MessageRead WsResponseStatus = "message.Read"

	MessageUpdate          WsResponseStatus = "message.Update"           // Добавить bool last_message_edited
	MessageClear           WsResponseStatus = "message.Clear"            // Добавить bool last_message_edited
	ChatNew                WsResponseStatus = "chat.New"                 // chat_information_dto
	ChatDeleted            WsResponseStatus = "chat.Deleted"             // chat_id
	ChatUpdatedAvatar      WsResponseStatus = "chat.Updated.Avatar"      // chat_id + avatar_url
	ChatUpdatedTitle       WsResponseStatus = "chat.Updated.Title"       // chat_id + title
	ChatUpdatedDescription WsResponseStatus = "chat.Updated.Description" // chat_id + description
	ChatUpdatedMembers     WsResponseStatus = "chat.Updated.Members"     // chat_id + type (deleted, added) + updated_members_id[] + name

	PresenceTyping   WsResponseStatus = "presence.Typing"
	PresenceOnline   WsResponseStatus = "presence.Online"
	PresenceOffline  WsResponseStatus = "presence.Offline"
	PresenceLastSeen WsResponseStatus = "presence.LastSeen"
)

type WsResponse[T any] struct {
	Payload T                `json:"payload"`
	Status  WsResponseStatus `json:"type"`
}
