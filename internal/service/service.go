package service

import (
	"context"
	"errors"
	"family-expenses/internal/entities"
	"family-expenses/internal/repository"
	"math"
)

type Service struct {
	Repo *repository.Repository
}

var ErrInvalidExpense = errors.New("invalid expense")

func (s *Service) ListExpenses(ctx context.Context, filter entities.ExpenseFilter) ([]entities.Expense, error) {
	return s.Repo.ListExpenses(ctx, filter)
}

func (s *Service) ListUserExpenses(ctx context.Context, userID int64) ([]entities.Expense, error) {
	return s.Repo.ListUserExpenses(ctx, userID)
}

func (s *Service) CreateUserExpense(ctx context.Context, exp *entities.Expense) error {
	if !validExpense(exp) {
		return ErrInvalidExpense
	}

	return s.Repo.CreateExpense(ctx, exp)
}

func (s *Service) UpdateUserExpense(ctx context.Context, exp *entities.Expense) error {
	if !validExpense(exp) {
		return ErrInvalidExpense
	}

	return s.Repo.UpdateUserExpense(ctx, exp)
}

func (s *Service) DeleteUserExpense(ctx context.Context, userID, expenseID int64) error {
	if userID <= 0 || expenseID <= 0 {
		return ErrInvalidExpense
	}

	return s.Repo.DeleteUserExpense(ctx, userID, expenseID)
}

func validExpense(exp *entities.Expense) bool {
	if exp == nil {
		return false
	}

	if exp.UserID <= 0 || math.IsNaN(exp.Amount) ||
		math.IsInf(exp.Amount, 0) || exp.Amount <= 0 ||
		exp.SpentAt.IsZero() {
		return false
	}

	switch exp.Category {
	case "еда", "транспорт", "жильё", "развлечения",
		"здоровье", "покупки", "другое":
		return true
	default:
		return false
	}
}

func (s *Service) UpdateExpense(ctx context.Context, exp *entities.Expense) error {
	if exp == nil || exp.ID <= 0 || !validExpense(exp) {
		return ErrInvalidExpense
	}

	return s.Repo.UpdateExpense(ctx, exp)
}

func (s *Service) DeleteExpense(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidExpense
	}

	return s.Repo.DeleteExpense(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context) ([]int64, error) {
	return s.Repo.ListUsers(ctx)
}
