package service

import (
	"bank/intern/domain"
	"bank/intern/repository"
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo       *repository.Repository
	jwtService *JWTService
}

func NewAuthService(repo *repository.Repository, jwtService *JWTService) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtService: jwtService,
	}
}

func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.repo.FindUserByLogin(ctx, login)
	if err != nil {
		return "", fmt.Errorf("failed to find user, %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidPassword
	}

	token, err := s.jwtService.GenerateToken(user)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}
