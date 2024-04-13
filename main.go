package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/playmixer/corvid/logger"
	tg "github.com/playmixer/telegram-bot-api"
	ygpt "github.com/playmixer/yandex/GPT"
)

type Store struct {
	message map[int64][]ygpt.YandexGPTMessage
	mu      sync.Mutex
}

func (s *Store) Set(key int64, value []ygpt.YandexGPTMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message[key] = value
}

func (s *Store) Get(key int64) []ygpt.YandexGPTMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.message[key]; ok {
		return v
	}
	return []ygpt.YandexGPTMessage{}
}

var (
	gpt *ygpt.YandexGPT
	// messages = make(map[int64][]ygpt.YandexGPTMessage)
	store *Store
	log   *logger.Logger
)

func init() {
	log = logger.New("logs")

	log.INFO("Init bot")

	err := godotenv.Load()
	if err != nil {
		log.ERROR("Error loading .env file")
	}

	gpt, err = ygpt.New(os.Getenv("YANDEX_API_KEY"), os.Getenv("YANDEX_FOLDER"))
	if err != nil {
		log.ERROR("Error init GPTChat, error:", err.Error())
	}

	store = &Store{
		message: make(map[int64][]ygpt.YandexGPTMessage),
		mu:      sync.Mutex{},
	}
}

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

	messages := store.Get(update.Message.Chat.Id)
	lastLen := max(len(messages)-19, 0)
	message := ygpt.YandexGPTMessage{Role: ygpt.GPTRoleUser, Text: update.Message.Text}
	store.Set(update.Message.Chat.Id, append(messages[lastLen:], message))

	req := gpt.NewRequest()
	messages = store.Get(update.Message.Chat.Id)
	b, _ := json.Marshal(messages)
	log.DEBUG(string(b))
	req.AddMessages(messages)
	req.CompletionOptions.Stream = true
	req.CompletionOptions.MaxTokens = 1000

	resp := make(chan ygpt.YandexGPTResponse, 1)
	err := req.DoStream(resp)
	if err != nil {
		log.ERROR(err.Error())
		bot.SendMessage(update.Message.Chat.Id, "Ошибка, попробуйте позже!")
	}

	for r := range resp {
		var assistentMessage ygpt.YandexGPTMessage
		for _, _resp := range r.Result.Alternatives {
			if _resp.Message.Text != "" {
				msg = bot.EditMessage(msg.Result.Chat.Id, msg.Result.MessageId, _resp.Message.Text)
				if !msg.Ok {
					log.WARN(msg.Description)
				}
			}
			assistentMessage = _resp.Message
		}
		store.Set(update.Message.Chat.Id, append(messages, assistentMessage))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	log.INFO("Starting...")
	bot, err := tg.NewBot(os.Getenv("TELEGRAM_BOT_API_KEY"))
	if err != nil {
		log.ERROR(err.Error())
		return
	}
	bot.AddHandle(tg.Command("start", start))
	bot.AddHandle(tg.Text(echo))

	bot.Timeout = time.Second
	bot.Polling()
	log.INFO("Stop")
	time.Sleep(time.Second)
}
