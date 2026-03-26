package contacts

import "errors"

var (
	ErrContactNotFound = errors.New("Contact not found")
	ErrContactsNotFound = errors.New("You dont have contacts with anyone")
	ErrCantCreateContactWithYourself = errors.New("You cant create contact with yourself")
	ErrContactExists = errors.New("Contact already exists")
)