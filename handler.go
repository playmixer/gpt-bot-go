package main

import (
	"encoding/json"
	"fmt"

	tg "github.com/playmixer/telegram-bot-api"
	ygpt "github.com/playmixer/yandex/GPT"
)

func start(update tg.UpdateResult, bot *tg.TelegramBot) {
	// fmt.Println(update.Message.Text)
	msg := bot.SendMessage(update.Message.Chat.Id, "Старт")
	if !msg.Ok {
		log.WARN(msg.Description)
	}
}

func echo(update tg.UpdateResult, bot *tg.TelegramBot) {
	msg := bot.ReplyToMessage(update.Message.Chat.Id, update.Message.MessageId, "...")
	if !msg.Ok {
		log.WARN(msg.Description)
	}

	bot.SendChatAction(msg.Result.Chat.Id, tg.TYPING)

	message := ygpt.YandexGPTMessage{Role: ygpt.GPTRoleUser, Text: update.Message.Text}
	store.Add(update.Message.Chat.Id, message, store.GetDefaultMessageLiveTime())

	req := gpt.NewRequest()
	messages := store.Get(update.Message.Chat.Id)
	// b, _ := json.Marshal(messages)
	log.DEBUG(fmt.Sprint("user messages", update.Message.Chat.Id, len(messages), messages))

	req.AddMessages(messages)
	req.CompletionOptions.Stream = true
	req.CompletionOptions.MaxTokens = 1000

	resp := make(chan ygpt.YandexGPTResponse, 1)
	err := req.DoStream(resp)
	if err != nil {
		log.ERROR(err.Error())
		bot.SendMessage(update.Message.Chat.Id, "Ошибка, попробуйте позже!")
	}

	var assistentMessage ygpt.YandexGPTMessage
	for r := range resp {
		b, _ := json.Marshal(r)
		log.DEBUG(fmt.Sprint(update.Message.Chat.Id), string(b))

		if r.StatusCode == 200 {
			for _, _resp := range r.Result.Alternatives {
				if _resp.Message.Text != "" {
					msg = bot.EditMessage(msg.Result.Chat.Id, msg.Result.MessageId, _resp.Message.Text)
					if !msg.Ok {
						log.WARN(msg.Description)
					}
				}
				assistentMessage = _resp.Message
			}
		}
		if r.StatusCode != 200 {

			msg = bot.EditMessage(msg.Result.Chat.Id, msg.Result.MessageId, fmt.Sprintf("Не удалось получить ответ, ошибка: %v %s %s", r.Error.HTTPCode, r.Error.HTTPStatus, r.Error.Message))
			if !msg.Ok {
				log.WARN(msg.Description)
			}
		}
	}
	store.Add(update.Message.Chat.Id, assistentMessage, store.GetDefaultMessageLiveTime())
}
