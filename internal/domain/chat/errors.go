package chat

import (
	"errors"
)

var (
	ErrChatNotFound                 = errors.New("Chat not found")
	ErrChatAlreadyExists            = errors.New("Chat already exists")
	ErrNoMessage                    = errors.New("No message in chat")
	ErrNotMember                    = errors.New("User is not member of this chat")
	ErrDialogAlreadyExists          = errors.New("dialog already exists between these users")
	ErrCantCreateDialogWithYourself = errors.New("you cant create dialog with yourself")
	ErrDialogMustHave2Users         = errors.New("dialog must have only 2 users")
	ErrOnlyOwnerCanDeleteChat       = errors.New("Only owner of the chat can delete chat")
)