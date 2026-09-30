package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrChatNotFound       = errors.New("chat not found")
	ErrForbidden          = errors.New("access denied")
)
