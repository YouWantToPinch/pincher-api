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

type Group struct {
	ID				uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	UserID     		uuid.UUID   `json:"user_id"`
	Name			string		`json:"name"`
	Notes			string		`json:"notes"`
}

type Category struct {
	ID				uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	UserID     		uuid.UUID   `json:"user_id"`
	Name			string		`json:"name"`
	GroupID			uuid.UUID	`json:"group_id"`
	Notes			string		`json:"notes"`
}
