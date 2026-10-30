package agent

import (
	"context"
	"family-expenses/internal/entities"
	"family-expenses/internal/llm"
	"family-expenses/internal/service"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type Agent struct {
	LLM     *llm.Client
	Service *service.Service
}

func (a *Agent) Process(ctx context.Context, userID int64, text string) (string, error) {
	action, err := a.LLM.Parse(ctx, text)
	if err != nil {
		slog.Error("Failed to parse Telegram message", "error", err)
		return "", err
	}

	switch action.Type {
	case "create":
		return a.create(ctx, userID, action)
	case "list":
		return a.list(ctx, userID, text, action)
	case "update":
		return a.update(ctx, userID, action)
	case "delete":
		return a.delete(ctx, userID, action)
	case "clarify":
		if action.Question != "" {
			return action.Question, nil
		}
		return "Уточни, пожалуйста, что нужно сделать?", nil
	default:
		return "Не понял запрос. Попробуй переформулировать.", nil
	}
}

func (a *Agent) create(ctx context.Context, userID int64, action *llm.Action) (string, error) {
	if action.Amount == nil || *action.Amount <= 0 || action.Category == "" {
		return "Укажи сумму и назначение траты, например: кофе 350.", nil
	}

	date := time.Now()
	if action.Date != "" {
		var err error
		date, err = time.Parse("2006-01-02", action.Date)
		if err != nil {
			return "Не понял дату траты.", nil
		}
	}

	exp := &entities.Expense{
		UserID:   userID,
		Amount:   *action.Amount,
		Category: action.Category,
		SpentAt:  date,
	}

	if action.Comment != "" {
		exp.Comment = &action.Comment
	}

	err := a.Service.CreateUserExpense(ctx, exp)
	if err != nil {
		if err == service.ErrInvalidExpense {
			return "Проверь сумму и категорию траты.", nil
		}
		return "", err
	}

	return "Записал: " + money(exp.Amount) + " ₽, " + exp.Category, nil
}

func (a *Agent) list(ctx context.Context, userID int64, text string, action *llm.Action) (string, error) {
	exps, err := a.Service.ListUserExpenses(ctx, userID)
	if err != nil {
		return "", err
	}

	var from, to time.Time

	if action.From != "" {
		from, err = time.Parse("2006-01-02", action.From)
		if err != nil {
			return "Не понял начало периода.", nil
		}
	}

	if action.To != "" {
		to, err = time.Parse("2006-01-02", action.To)
		if err != nil {
			return "Не понял конец периода.", nil
		}
	}

	var lines []string
	var sum float64
	var count int

	for _, exp := range exps {
		if action.Category != "" && exp.Category != action.Category {
			continue
		}

		if action.Search != "" {
			name := strings.ToLower(exp.Category)
			if exp.Comment != nil {
				name += " " + strings.ToLower(*exp.Comment)
			}

			if !strings.Contains(name, strings.ToLower(action.Search)) {
				continue
			}
		}

		if !from.IsZero() && exp.SpentAt.Before(from) {
			continue
		}

		if !to.IsZero() && exp.SpentAt.After(to) {
			continue
		}

		count++
		sum += exp.Amount

		if len(lines) < 100 {
			lines = append(lines, expenseLine(exp))
		}
	}
	if len(lines) >= 100 {
		lines = append(lines, "(Слишком много строк. Показаны первые 100 трат)")
	}

	if count == 0 {
		return "Расходов не нашёл.", nil
	}

	result := "Всего: " + money(sum) + " ₽ (" + strconv.Itoa(count) + " трат)."

	return result + "\n\n" + strings.Join(lines, "\n"), nil
}

func (a *Agent) update(ctx context.Context, userID int64, action *llm.Action) (string, error) {
	exp, answer, err := a.findExpense(ctx, userID, action)
	if err != nil || exp == nil {
		return answer, err
	}

	changed := false

	if action.Amount != nil {
		exp.Amount = *action.Amount
		changed = true
	}

	if action.Category != "" {
		exp.Category = action.Category
		changed = true
	}

	if action.Comment != "" {
		exp.Comment = &action.Comment
		changed = true
	}

	if action.Date != "" {
		date, err := time.Parse("2006-01-02", action.Date)
		if err != nil {
			return "Не понял новую дату.", nil
		}
		exp.SpentAt = date
		changed = true
	}

	if !changed {
		return "Что именно нужно изменить?", nil
	}

	err = a.Service.UpdateUserExpense(ctx, exp)
	if err != nil {
		if err == service.ErrInvalidExpense {
			return "Некорректные данные траты.", nil
		}
		return "", err
	}

	return "Изменил: " + expenseLine(*exp), nil
}

func (a *Agent) delete(ctx context.Context, userID int64, action *llm.Action) (string, error) {
	exp, answer, err := a.findExpense(ctx, userID, action)
	if err != nil || exp == nil {
		return answer, err
	}

	err = a.Service.DeleteUserExpense(ctx, userID, exp.ID)
	if err != nil {
		return "", err
	}

	return "Удалил: " + expenseLine(*exp), nil
}

func (a *Agent) findExpense(ctx context.Context, userID int64, action *llm.Action) (*entities.Expense, string, error) {
	exps, err := a.Service.ListUserExpenses(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	if action.ExpenseID != nil {
		for _, exp := range exps {
			if exp.ID == *action.ExpenseID {
				return &exp, "", nil
			}
		}
		return nil, "Трату с таким ID не нашёл.", nil
	}

	search := strings.ToLower(strings.TrimSpace(action.Search))

	if search == "" {
		return nil, "Укажи название траты или её ID.", nil
	}

	if search == "последнее" ||
		search == "последний" ||
		search == "последнюю" ||
		search == "последней" {
		if len(exps) == 0 {
			return nil, "У тебя пока нет расходов.", nil
		}
		return &exps[0], "", nil
	}

	var matches []entities.Expense

	for _, exp := range exps {
		name := strings.ToLower(exp.Category)

		if exp.Comment != nil {
			name += " " + strings.ToLower(*exp.Comment)
		}

		if strings.Contains(name, search) {
			matches = append(matches, exp)
		}
	}

	if len(matches) == 0 {
		return nil, "Такую трату не нашёл.", nil
	}

	if len(matches) > 1 {
		var lines []string

		for i, exp := range matches {
			if i >= 10 {
				break
			}
			lines = append(lines, expenseLine(exp))
		}

		return nil, "Нашёл несколько трат. Укажи ID нужной:\n" +
			strings.Join(lines, "\n"), nil
	}

	return &matches[0], "", nil
}

func expenseLine(exp entities.Expense) string {
	name := exp.Category

	if exp.Comment != nil && *exp.Comment != "" {
		name = *exp.Comment
	}

	return "#" + strconv.FormatInt(exp.ID, 10) +
		" · " + exp.SpentAt.Format("02.01.2006") +
		" · " + name +
		" · " + money(exp.Amount) + " ₽"
}

func money(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}
