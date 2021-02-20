package lemon_api

import "time"

type User struct {
	ID        string `json:"id" db:"id"`
	Username  string `json:"username" db:"username"`
	Hash      string `json:"hash" db:"hash"`
	SaveState string `json:"save_state" db:"save_state"`
}

type Feedback struct {
	ID          int64      `json:"id" db:"id"`
	Rating      int64      `json:"rating" db:"rating"`
	Description string     `json:"description" db:"description"`
	Type        string     `json:"type" db:"type"`
	Submitted   *time.Time `json:"submitted" db:"submitted"`
	Read        bool       `json:"read" db:"read"`
}

type TokenRequest struct {
	Username string `json:"username"`
	Hash     string `json:"hash"`
}

type Token struct {
	Value string `json:"token"`
}
