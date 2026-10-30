package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	URL    string
	Model  string
	APIKey string
	HTTP   *http.Client
}

type Action struct {
	Type      string   `json:"action"`
	Amount    *float64 `json:"amount"`
	Category  string   `json:"category"`
	Comment   string   `json:"comment"`
	Date      string   `json:"date"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Search    string   `json:"search"`
	ExpenseID *int64   `json:"expense_id"`
	Question  string   `json:"question"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"max_tokens"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func NewClient(url, model, apiKey string) *Client {
	return &Client{
		URL:    strings.TrimRight(url, "/"),
		Model:  model,
		APIKey: apiKey,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Parse(ctx context.Context, text string) (*Action, error) {
	prompt := `Ты анализируешь сообщения учёта расходов.
Верни только JSON, без пояснений.
Действия: create, list, update, delete, clarify.
Категории: еда, транспорт, жильё, развлечения, здоровье, покупки, другое.
Поля JSON: action, amount, category, comment, date, from, to, search, expense_id, question.
Для create:
	amount — сумма расхода.
	category — категория из списка.
	comment — что именно купил пользователь.
	date — дата в формате YYYY-MM-DD.
Для list заполни category, from и to, если есть соответствующие ограничения.
Для update и delete заполни search для поиска нужной траты, а amount, category, comment, date — только если пользователь хочет изменить соответствующее поле.
Если пользователь говорит "последнее", запиши "последнее" в search.
expense_id заполняй только если пользователь явно указал числовой ID записи. Не придумывай ID.
Если информации недостаточно для определения действия, используй clarify и напиши вопрос в question.
Не придумывай сумму и другие данные.
Отсутствующие значения: null для amount и expense_id, пустая строка для остальных полей.
Даты записывай в формате YYYY-MM-DD. Сегодня: ` + time.Now().Format("2006-01-02")

	payload := chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: text},
		},
		Temperature: 0,
		MaxTokens:   1800,
	}
	payload.ResponseFormat.Type = "json_object"

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal LLM request", "error", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.URL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		slog.Error("Failed to create LLM request", "error", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		slog.Error("Failed to call LLM", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		slog.Error("LLM returned an error",
			"status", resp.StatusCode,
			"body", string(data),
		)
		return nil, errors.New("LLM request failed")
	}

	var result chatResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		slog.Error("Failed to decode LLM response", "error", err)
		return nil, err
	}

	if len(result.Choices) == 0 {
		err = errors.New("empty LLM response")
		slog.Error("LLM returned no choices", "error", err)
		return nil, err
	}

	content := result.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		err = errors.New("empty LLM content")
		slog.Error("LLM returned empty content", "error", err)
		return nil, err
	}

	var action Action
	err = json.Unmarshal([]byte(content), &action)
	if err != nil {
		slog.Error("Failed to parse LLM action",
			"error", err,
			"content", content,
		)
		return nil, err
	}

	switch action.Type {
	case "create", "list", "update", "delete", "clarify":
	default:
		err = errors.New("unknown LLM action")
		slog.Error("LLM returned unknown action", "action", action.Type)
		return nil, err
	}

	slog.Info("LLM parsed action", "action", action.Type)

	return &action, nil
}
