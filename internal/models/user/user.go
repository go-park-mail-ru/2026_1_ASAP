package models

type User struct {
	Id           int    `json:"id"`
	Login        string `json:"login"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}
