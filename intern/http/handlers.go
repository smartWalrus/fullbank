package http

import (
	dto "bank/intern/DTO"
	"bank/intern/domain"
	"bank/intern/service"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	service *service.Service
	auth    *service.AuthService
}

func CreateHandlers(serv *service.Service, auth *service.AuthService) *Handlers {
	return &Handlers{service: serv, auth: auth}
}

func (h *Handlers) CreateUser(c *gin.Context) {
	var user dto.UserDTO
	if err := c.BindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	err := h.service.CreateUser(c.Request.Context(), user.Login, user.Password, user.Role)
	if err != nil {
		if errors.Is(err, domain.ErrLoginAlreadyExists) {
			c.JSON(409, gin.H{"error": "login already exists"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "user created"})
}
func (h *Handlers) CreateCard(c *gin.Context) {
	var card struct {
		Phone string `json:"phone"`
	}
	if err := c.BindJSON(&card); err != nil {
		c.JSON(400, gin.H{"error": "failed to parse phone"})
		return
	}
	err := h.service.CreateCard(c.Request.Context(), card.Phone)
	if err != nil {
		if errors.Is(err, domain.ErrFailedToValidatePhone) {
			c.JSON(500, gin.H{"error": "validation failed"})
			return
		}
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "card succesfully added"})
}
func (h *Handlers) RentCard(c *gin.Context) {
	var req struct {
		Login      string `json:"login"`
		Hours      int    `json:"hours"`
		CardAmount int    `json:"amount"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	amount, err := h.service.RentCard(c.Request.Context(), req.Login, req.CardAmount, req.Hours)
	if err != nil {

		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(404, gin.H{"error": "user not found. try another login"})
			return
		}

		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	message := fmt.Sprintf("for user - %s has been added %d cards\n", req.Login, amount)
	c.JSON(200, gin.H{"message": message})
}
func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPassword) {
			c.JSON(401, gin.H{"error": "try another password"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"token": token})
}
func (h *Handlers) GetActiveCards(c *gin.Context) {
	idIn, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, ok := idIn.(int64)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid user id type"})
		return
	}
	cards, err := h.service.GetActiveCards(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNoActiveCards) {
			c.JSON(404, gin.H{"error": "no active cards available"})
			return
		}
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}
	c.JSON(200, cards)
}
func (h *Handlers) GetExpiredCards(c *gin.Context) {
	idIn, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, ok := idIn.(int64)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid user id type"})
		return
	}
	cards, err := h.service.GetExpiredCards(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNoExpiredCards) {
			c.JSON(404, gin.H{"error": "no expired cards available"})
			return
		}
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}
	c.JSON(200, cards)
}
func (h *Handlers) UpdatePass(c *gin.Context) {
	var req struct {
		OldPass string `json:"oldPass"`
		NewPass string `json:"newPass"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	login, exists := c.Get("login")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	loginStr, ok := login.(string)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid login type"})
		return
	}
	if err := h.service.ChangePass(c.Request.Context(), loginStr, req.OldPass, req.NewPass); err != nil {
		if errors.Is(err, domain.ErrInvalidPassword) {
			c.JSON(400, gin.H{"error": domain.ErrInvalidPassword.Error()})
			return
		}
		if errors.Is(err, domain.ErrSamePass) {
			c.JSON(400, gin.H{"error": domain.ErrSamePass.Error()})
			return
		}
		if errors.Is(err, domain.ErrWeakPass) {
			c.JSON(400, gin.H{"error": domain.ErrWeakPass.Error()})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "update succeded"})
}
func (h *Handlers) WriteNote(c *gin.Context) {
	var req struct {
		CardId  int    `json:"card_id"`
		Message string `json:"message"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "failed to parse data"})
		return
	}
	if err := h.service.WriteNote(c.Request.Context(), req.CardId, req.Message); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "comment added"})
}
