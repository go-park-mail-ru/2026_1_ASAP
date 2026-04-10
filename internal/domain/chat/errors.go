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
	ErrDialogCannotHaveCustomAvatar = errors.New("Cant change avatar in dialog")
	ErrDialogCannotHaveCustomTitle  = errors.New("Cant change title in dialog")
	ErrOnlyOwnerCanAddPeople        = errors.New("Only owner of chat can add people to chat")
	ErrMemberAlreadyInChat          = errors.New("Member you try to add already in chat")
	ErrCantAddMemberToDialog        = errors.New("You cant add member in your dialog chat")
	ErrMemberNotFound               = errors.New("Not found member you want to delete")
	ErrOnlyOwnerCanDeletePeople     = errors.New("Only owner of chat can delete people from chat")
	ErrCantDeleteMemberFromDialog   = errors.New("You cant delete member from your dialog chat")
	ErrCantDeleteOwnerOfChat        = errors.New("You cant delete owner of chat from the chat")
	ErrUserNotMember                = errors.New("User you try to delete is not member of this chat")
)