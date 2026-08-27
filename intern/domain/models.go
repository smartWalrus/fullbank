package domain

import "time"

type User struct {
	Id           int64
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	Role         string
}
type VerificationCode struct {
	Id        int64
	UserId    int64
	Phone     string
	Code      string
	ExpiresAt time.Time
	IsUsed    bool
}
type SimCard struct {
	Id        int64
	Phone     string
	CreatedAt time.Time
	IsActive  bool
	Message   string
}
type Rental struct {
	Id        int64     `json:"id"`
	CardId    int64     `json:"card_id"`
	UserId    int64     `json:"user_id"`
	Phone     string    `json:"phone"`
	RentedAt  time.Time `json:"rented_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
}
