package dto

type UserDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}
