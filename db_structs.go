package main

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        		uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	Username     	string   	`json:"username"`
	HashedPassword	string		`json:"hashed_password"`
	Token			string		`json:"token"`
	RefreshToken	string		`json:"refresh_token"`
}