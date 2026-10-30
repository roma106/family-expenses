package repository

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

var (
	ErrNotFound = errors.New("Expense not found")
)
