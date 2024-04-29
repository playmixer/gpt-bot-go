package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	ygpt "github.com/playmixer/yandex/GPT"
)

var (
	COUNT_STORE_MESSAGE = 10
)

type StoreMessage struct {
	Expired time.Time
	Message ygpt.YandexGPTMessage
}

type Store struct {
	Data                   map[int64][]StoreMessage
	SystemMessage          ygpt.YandexGPTMessage
	DefaultMessageLiveTime time.Duration
	CountStoreMessage      int
	ctx                    context.Context
	mu                     sync.Mutex
}

type Option func(s *Store)

func New(ctx context.Context, options ...Option) *Store {
	s := &Store{
		ctx:                    ctx,
		DefaultMessageLiveTime: time.Hour * 24,
		Data:                   make(map[int64][]StoreMessage),
		SystemMessage:          ygpt.YandexGPTMessage{},
		CountStoreMessage:      COUNT_STORE_MESSAGE,
		mu:                     sync.Mutex{},
	}
	for _, opt := range options {
		opt(s)
	}
	// go s.garbage()
	return s
}

func OptionMessageLiveTime(liveTime time.Duration) func(s *Store) {
	if liveTime < time.Minute {
		liveTime = time.Minute
	}
	return func(s *Store) {
		s.DefaultMessageLiveTime = liveTime
	}
}

func OptionCountStoreMessage(count int) func(s *Store) {
	return func(s *Store) {
		s.CountStoreMessage = max(count, 2)
	}
}

func (s *Store) Set(key int64, value []StoreMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

func (s *Store) Add(key int64, value ygpt.YandexGPTMessage, liveTime time.Duration) {
	if _, ok := s.Data[key]; !ok {
		s.Data[key] = []StoreMessage{}
	}
	startLen := s.CountStoreMessage
	if len(s.Data[key]) < startLen {
		startLen = max(len(s.Data[key])-s.CountStoreMessage, 0)
	}
	s.Data[key] = append(s.Data[key][startLen:], StoreMessage{
		Expired: time.Now().Add(liveTime),
		Message: value,
	})
}

func (s *Store) GetDefaultMessageLiveTime() time.Duration {
	return s.DefaultMessageLiveTime
}

func (s *Store) SetSystem(value ygpt.YandexGPTMessage) {
	s.SystemMessage = value
}

func (s *Store) Get(key int64) []ygpt.YandexGPTMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := []StoreMessage{}
	yMsg := []ygpt.YandexGPTMessage{}
	fmt.Println("store get", s.Data[key])
	if v, ok := s.Data[key]; ok {
		for _, _msg := range v {
			if time.Until(_msg.Expired) > 0 {
				msg = append(msg, _msg)
				yMsg = append(yMsg, _msg.Message)
			}
		}
	}
	startMessage := max(len(msg)-s.CountStoreMessage, 0)
	s.Data[key] = msg[startMessage:]

	if s.SystemMessage.Text != "" {
		return append([]ygpt.YandexGPTMessage{s.SystemMessage}, yMsg...)
	}
	return yMsg
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
