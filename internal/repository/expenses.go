package repository

import (
	"context"
	"database/sql"
	"errors"
	"family-expenses/internal/entities"
	"log/slog"
)

func (repo *Repository) ListExpenses(ctx context.Context, filter entities.ExpenseFilter) ([]entities.Expense, error) {
	exps := []entities.Expense{}

	err := repo.DB.SelectContext(ctx, &exps, `
        SELECT * FROM expenses
        WHERE ($1::bigint IS NULL OR user_id = $1)
          AND ($2::text = '' OR category = $2)
          AND ($3::date IS NULL OR spent_at >= $3)
          AND ($4::date IS NULL OR spent_at <= $4)
        ORDER BY spent_at DESC, created_at DESC, id DESC
    `, filter.UserID, filter.Category, filter.From, filter.To)

	if err != nil {
		slog.Error("Failed to get expenses", "error", err)
		return nil, err
	}

	return exps, nil
}

func (repo *Repository) ListUserExpenses(ctx context.Context, userID int64) ([]entities.Expense, error) {
	exps := []entities.Expense{}

	err := repo.DB.SelectContext(ctx, &exps, `
		SELECT * FROM expenses
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`, userID)

	if err != nil {
		slog.Error("Failed to get user expenses", "error", err)
		return nil, err
	}

	return exps, nil
}

func (repo *Repository) CreateExpense(ctx context.Context, exp *entities.Expense) error {
	err := repo.DB.GetContext(ctx, exp, `
		INSERT INTO expenses (user_id, amount, category, spent_at, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *
	`, exp.UserID, exp.Amount, exp.Category, exp.SpentAt, exp.Comment)

	if err != nil {
		slog.Error("Failed to create expense", "error", err)
		return err
	}

	return nil
}

func (repo *Repository) UpdateUserExpense(ctx context.Context, exp *entities.Expense) error {
	err := repo.DB.GetContext(ctx, exp, `
		UPDATE expenses
		SET amount = $1, category = $2, spent_at = $3,
		    comment = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING *
	`, exp.Amount, exp.Category, exp.SpentAt, exp.Comment, exp.ID, exp.UserID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	if err != nil {
		slog.Error("Failed to update expense", "error", err)
		return err
	}

	return nil
}

func (repo *Repository) DeleteUserExpense(ctx context.Context, userID, expenseID int64) error {
	result, err := repo.DB.ExecContext(ctx,
		"DELETE FROM expenses WHERE id = $1 AND user_id = $2",
		expenseID, userID,
	)

	if err != nil {
		slog.Error("Failed to delete expense", "error", err)
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get affected rows", "error", err)
		return err
	}

	if count == 0 {
		return ErrNotFound
	}

	return nil
}

func (repo *Repository) UpdateExpense(ctx context.Context, exp *entities.Expense) error {
	err := repo.DB.GetContext(ctx, exp, `
        UPDATE expenses
        SET user_id = $1, amount = $2, category = $3,
            spent_at = $4, comment = $5, updated_at = NOW()
        WHERE id = $6
        RETURNING *
    `, exp.UserID, exp.Amount, exp.Category,
		exp.SpentAt, exp.Comment, exp.ID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	if err != nil {
		slog.Error("Failed to update expense", "error", err)
		return err
	}

	return nil
}

func (repo *Repository) DeleteExpense(ctx context.Context, id int64) error {
	result, err := repo.DB.ExecContext(ctx,
		"DELETE FROM expenses WHERE id = $1", id)

	if err != nil {
		slog.Error("Failed to delete expense", "error", err)
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to count deleted expenses", "error", err)
		return err
	}

	if count == 0 {
		return ErrNotFound
	}

	return nil
}

func (repo *Repository) ListUsers(ctx context.Context) ([]int64, error) {
	users := []int64{}

	err := repo.DB.SelectContext(ctx, &users, `
        SELECT DISTINCT user_id
        FROM expenses
        ORDER BY user_id
    `)

	if err != nil {
		slog.Error("Failed to get users", "error", err)
		return nil, err
	}

	return users, nil
}
