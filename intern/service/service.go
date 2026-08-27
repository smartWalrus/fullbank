package service

import (
	"bank/intern/domain"
	"bank/intern/metrics"
	"bank/intern/repository"
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *repository.Repository
}

func CreateService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}
func (s *Service) CreateUser(ctx context.Context, login, password, role string) error {
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		return fmt.Errorf("hash password: %w", err)
	}
	user := domain.User{
		Login:        login,
		PasswordHash: string(hashPass),
		CreatedAt:    time.Now(),
		Role:         role,
	}
	if err := s.repo.CreateUser(ctx, &user); err != nil {
		slog.Error("Failed to create user", "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}
	metrics.UsersRegisteredTotal.Inc()
	return nil
}

func (s *Service) CreateCard(ctx context.Context, phone string) error {
	card := &domain.SimCard{
		Phone:     phone,
		CreatedAt: time.Now(),
	}
	if err := validatePhone(phone); err != nil {
		slog.Warn("Phone validation failed", "phone", phone, "error", err)
		return fmt.Errorf("validation failed, %w", err)
	}
	if err := s.repo.CreateCard(ctx, card); err != nil {
		slog.Error("Failed to create card", "phone", phone, "error", err)
		return fmt.Errorf("create card: %w", err)
	}
	slog.Info("Card created", "phone", phone)
	metrics.CardsTotal.Inc()
	metrics.CardsFree.Inc()
	return nil
}
func (s *Service) RentCard(ctx context.Context, login string, cardAmount, hours int) (int, error) {
	user, err := s.repo.FindUserByLogin(ctx, login)
	if err != nil {
		slog.Warn("User not found for rent", "login", login, "error", err)
		return 0, fmt.Errorf("failed to find user, %w", err)

	}
	ids, err := s.repo.FindFreeCards(ctx, cardAmount)

	count := len(ids)
	if count < cardAmount {
		slog.Warn("Not enough free cards", "requested", cardAmount, "available", count)
		return count, fmt.Errorf("only %d cards available", count)
	}
	if err != nil {
		slog.Error("Failed to find free cards", "login", login, "error", err)
		return 0, fmt.Errorf("failed to find free cards, %w", err)
	}
	expirationTime := time.Now().Add(time.Duration(hours) * time.Hour)

	if err := s.repo.RentCard(ctx, user, ids, expirationTime); err != nil {
		slog.Error("Failed to rent card", "login", login, "error", err)
		return 0, fmt.Errorf("failed to rent a card, %w", err)
	}
	slog.Info("Cards rented", "login", login, "count", count)
	metrics.RentalsCreatedTotal.Inc()
	metrics.RentalsActive.Inc()
	metrics.CardsFree.Sub(float64(len(ids)))
	metrics.CardsBusy.Add(float64(len(ids)))
	return count, nil
}

func (s *Service) Login(ctx context.Context, login, pass string) error {
	user, err := s.repo.FindUserByLogin(ctx, login)
	if err != nil {
		slog.Warn("Login failed: user not found", "login", login, "error", err)
		return fmt.Errorf("failed to find user, %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(pass))
	if err != nil {
		slog.Warn("Login failed: invalid password", "login", login)
		return fmt.Errorf("incorrect pass, %w", err)
	}
	slog.Info("User logged in", "login", login)
	return nil
}
func (s *Service) ChangePass(ctx context.Context, login, oldPass, pass string) error {
	if len(pass) < 8 {
		slog.Warn("Password too weak", "login", login)
		return domain.ErrWeakPass
	}

	user, err := s.repo.FindUserByLogin(ctx, login)
	if err != nil {
		slog.Warn("Change password: user not found", "login", login, "error", err)
		return fmt.Errorf("failed to find user by login, %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPass)); err != nil {
		slog.Warn("Change password: invalid old password", "login", login)
		return fmt.Errorf("failed to check the old pass, %w", domain.ErrInvalidPassword)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(pass)); err == nil {
		slog.Warn("Change password: new password same as old", "login", login)
		return domain.ErrSamePass
	}
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Change password: failed to hash", "login", login, "error", err)
		return fmt.Errorf("failed to hash password, %w", err)
	}
	user.PasswordHash = string(hashedPass)
	if err := s.repo.UpdatePass(ctx, user); err != nil {
		slog.Error("Change password: failed to update in DB", "login", login, "error", err)
		return fmt.Errorf("failed to update pass, %w", err)
	}
	slog.Info("Password changed", "login", login)
	return nil
}

func (s *Service) StartExpirationChecker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			count, err := s.repo.ExpireRentals(ctx)
			if err != nil {
				slog.Error("Expire rentals failed", "error", err)
				continue
			}
			if count > 0 {
				metrics.RentalsActive.Sub(float64(count))
				metrics.CardsBusy.Sub(float64(count))
				metrics.CardsFree.Add(float64(count))
				slog.Info("Rentals expired", "count", count)
			}
		}
	}()
}
func (s *Service) GetExpiredCards(ctx context.Context, id int64) ([]*domain.Rental, error) {

	cards, err := s.repo.GetExpiredCards(ctx, id)
	if err != nil {
		slog.Error("Failed to get expired cards", "user_id", id, "error", err)
		return nil, fmt.Errorf("get expired cards: %w", err)
	}
	if len(cards) == 0 {
		slog.Warn("No expired cards for user", "user_id", id)
		return nil, domain.ErrNoExpiredCards
	}
	slog.Info("Got expired cards", "user_id", id, "count", len(cards))
	return cards, nil
}

func (s *Service) GetActiveCards(ctx context.Context, id int64) ([]*domain.Rental, error) {

	cards, err := s.repo.GetActiveCards(ctx, id)
	if err != nil {
		slog.Error("Failed to get active cards", "user_id", id, "error", err)
		return nil, fmt.Errorf("get active cards: %w", err)
	}
	if len(cards) == 0 {
		slog.Warn("No active cards for user", "user_id", id)
		return nil, domain.ErrNoActiveCards
	}
	slog.Info("Got active cards", "user_id", id, "count", len(cards))
	return cards, nil
}
func validatePhone(phone string) error {
	time.Sleep(1 * time.Second)
	return nil
}
func (s *Service) WriteNote(ctx context.Context, id int, message string) error {
	if err := s.repo.WriteNote(ctx, id, message); err != nil {
		return err
	}
	return nil
}
