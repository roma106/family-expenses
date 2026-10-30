package tg

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler func(
	ctx context.Context,
	userID int64,
	text string,
) (string, error)

type Bot struct {
	api     *tgbotapi.BotAPI
	handler MessageHandler
}

func NewBot(token string, handler MessageHandler) (*Bot, error) {
	if token == "" || handler == nil {
		slog.Error("Failed to create bot: bad input")
		return nil, fmt.Errorf("token or handler is nil")
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		return nil, err
	}

	return &Bot{
		api:     api,
		handler: handler,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := b.api.GetUpdatesChan(updateConfig)

	slog.Info("Telegram bot started")

	defer b.api.StopReceivingUpdates()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Telegram bot stopped")
			return

		case update, ok := <-updates:
			if !ok {
				slog.Error("Error occured while gettin message")
				return
			}

			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	message := update.Message

	if message == nil || message.From == nil {
		return
	}

	if !message.Chat.IsPrivate() {
		return
	}

	text := strings.TrimSpace(message.Text)

	if text == "" {
		b.sendMessage(
			message.Chat.ID,
			"Пока я понимаю только текстовые сообщения.",
		)
		return
	}

	userID := message.From.ID

	response, err := b.handler(ctx, userID, text)
	if err != nil {
		slog.Error("Failed to process Telegram message",
			"user_id", userID,
			"error", err,
		)

		b.sendMessage(
			message.Chat.ID,
			"Не получилось обработать сообщение. Попробуй ещё раз.",
		)
		return
	}

	if response == "" {
		response = "Готово!"
	}

	b.sendMessage(message.Chat.ID, response)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := b.api.Send(msg)
	if err != nil {
		slog.Error("Failed to send Telegram message",
			"chat_id", chatID,
			"error", err,
		)
	}
}
