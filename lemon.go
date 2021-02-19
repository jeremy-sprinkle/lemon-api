package lemon_api

import "time"

type Lemon struct {
	Thing string `json:"thing" db:"thing"`
}

type Feedback struct {
	ID          int64      `json:"id" db:"id"`
	Rating      int64      `json:"rating" db:"rating"`
	Description string     `json:"description" db:"description"`
	Type        string     `json:"type" db:"type"`
	Submitted   *time.Time `json:"submitted" db:"submitted"`
	Read        bool       `json:"read" db:"read"`
}
