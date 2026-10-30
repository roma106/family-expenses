package handler

import (
	"errors"
	"family-expenses/internal/entities"
	"family-expenses/internal/repository"
	"family-expenses/internal/service"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type expenseRequest struct {
	UserID   int64   `json:"user_id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	SpentAt  string  `json:"spent_at"`
	Comment  string  `json:"comment"`
}

func (h *Handler) ListExpenses(c echo.Context) error {
	filter := entities.ExpenseFilter{
		Category: c.QueryParam("category"),
	}

	if value := c.QueryParam("user_id"); value != "" {
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if userID <= 0 {
			return c.String(http.StatusBadRequest, "Invalid user ID")
		}

		filter.UserID = &userID
	}

	if value := c.QueryParam("from"); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		filter.From = &date
	}

	if value := c.QueryParam("to"); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		filter.To = &date
	}

	if filter.From != nil && filter.To != nil {
		if filter.From.After(*filter.To) {
			return c.String(http.StatusBadRequest, "Invalid date range")
		}
	}

	exps, err := h.Service.ListExpenses(c.Request().Context(), filter)
	if err != nil {
		slog.Error("Failed to list expenses", "error", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, exps)
}

func (h *Handler) ListUsers(c echo.Context) error {
	users, err := h.Service.ListUsers(c.Request().Context())
	if err != nil {
		slog.Error("Failed to list users", "error", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, users)
}

func (h *Handler) CreateExpense(c echo.Context) error {
	exp, err := readExpense(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "Некорректный JSON или дата"})
	}

	err = h.Service.CreateUserExpense(c.Request().Context(), exp)
	if err != nil {
		return expenseError(c, err)
	}

	return c.JSON(http.StatusCreated, exp)
}

func (h *Handler) UpdateExpense(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "Некорректный ID"})
	}

	exp, err := readExpense(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "Некорректный JSON или дата"})
	}

	exp.ID = id

	err = h.Service.UpdateExpense(c.Request().Context(), exp)
	if err != nil {
		return expenseError(c, err)
	}

	return c.JSON(http.StatusOK, exp)
}

func (h *Handler) DeleteExpense(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "Некорректный ID"})
	}

	err = h.Service.DeleteExpense(c.Request().Context(), id)
	if err != nil {
		return expenseError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func readExpense(c echo.Context) (*entities.Expense, error) {
	var input expenseRequest

	if err := c.Bind(&input); err != nil {
		slog.Error("Failed to read expense request", "error", err)
		return nil, err
	}

	date, err := time.Parse("2006-01-02", input.SpentAt)
	if err != nil {
		slog.Error("Failed to parse expense date", "error", err)
		return nil, err
	}

	exp := &entities.Expense{
		UserID:   input.UserID,
		Amount:   input.Amount,
		Category: input.Category,
		SpentAt:  date,
	}

	if comment := strings.TrimSpace(input.Comment); comment != "" {
		exp.Comment = &comment
	}

	return exp, nil
}

func expenseError(c echo.Context, err error) error {
	if errors.Is(err, service.ErrInvalidExpense) {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "Некорректные данные расхода"})
	}

	if errors.Is(err, repository.ErrNotFound) {
		return c.JSON(http.StatusNotFound,
			map[string]string{"error": "Расход не найден"})
	}

	slog.Error("Failed to process expense", "error", err)

	return c.String(http.StatusInternalServerError, err.Error())
}
