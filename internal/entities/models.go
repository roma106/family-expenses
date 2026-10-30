package entities

import "time"

type Expense struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Amount    float64   `db:"amount" json:"amount"`
	Category  string    `db:"category" json:"category"`
	SpentAt   time.Time `db:"spent_at" json:"spent_at"`
	Comment   *string   `db:"comment" json:"comment"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ExpenseFilter struct {
	UserID   *int64
	Category string
	From     *time.Time
	To       *time.Time
}
