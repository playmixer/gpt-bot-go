package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"gpt-telegram-bot/storage"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/playmixer/corvid/logger"
	tg "github.com/playmixer/telegram-bot-api"
	ygpt "github.com/playmixer/yandex/GPT"
)

type StorageInterface interface {
	Add(key int64, value ygpt.YandexGPTMessage, liveTime time.Duration)
	SetSystem(value ygpt.YandexGPTMessage)
	Get(key int64) []ygpt.YandexGPTMessage
	GetDefaultMessageLiveTime() time.Duration
}

var (
	gpt   *ygpt.YandexGPT
	store StorageInterface
	log   *logger.Logger
)

func init() {
	log = logger.New("logs")
	log.INFO("Init bot")

	err := godotenv.Load()
	if err != nil {
		log.ERROR("Error loading .env file")
	}

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logger.DEBUG)
		log.INFO("debug level")
	} else {
		log.SetLevel(logger.INFO)
	}

	gpt, err = ygpt.New(os.Getenv("YANDEX_API_KEY"), os.Getenv("YANDEX_FOLDER"))
	if err != nil {
		log.ERROR("Error init GPTChat, error:", err.Error())
	}

}

func main() {
	log.INFO("Starting...")
	ctx, cancel := context.WithCancel(context.Background())

	if os.Getenv("TLS") == "0" {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	livetime, _ := strconv.Atoi(os.Getenv("MSG_LIVE_TIME"))
	countStoreMessage, _ := strconv.Atoi(os.Getenv("COUNT_STORE_MESSAGE"))
	store = storage.New(
		ctx,
		storage.OptionMessageLiveTime(time.Minute*time.Duration(livetime)),
		storage.OptionCountStoreMessage(countStoreMessage),
	)
	store.SetSystem(ygpt.YandexGPTMessage{Role: ygpt.GPTRoleSystem, Text: "Ты умный ассистент"})

	fmt.Println(store.GetDefaultMessageLiveTime())
	fmt.Println(countStoreMessage)

	bot, err := tg.NewBot(os.Getenv("TELEGRAM_BOT_API_KEY"))
	if err != nil {
		log.ERROR(err.Error())
		return
	}
	bot.AddHandle(tg.Command("start", start))
	bot.AddHandle(tg.Text(echo))

	bot.Timeout = time.Second
	bot.Polling()
	cancel()
	log.INFO("Stop")
	time.Sleep(time.Second)
}
