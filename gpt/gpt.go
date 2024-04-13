package gpt

import (
	"context"
	"fmt"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

const (
	MSG_LIMIT int = 5
)

var (
	GPTClient *Gpt
)

type Gpt struct {
	Client  *openai.Client
	Message []openai.ChatCompletionMessage
	mu      sync.Mutex
}

func Init(token string) *Gpt {
	GPTClient = &Gpt{
		Client:  openai.NewClient(token),
		Message: make([]openai.ChatCompletionMessage, 0),
	}

	return GPTClient
}

func (g *Gpt) Request(text string) (string, error) {
	g.mu.Lock()

	g.Message = append(g.Message, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: text,
	})
	count_message := len(g.Message)
	if count_message > MSG_LIMIT {
		g.Message = g.Message[count_message-MSG_LIMIT:]
	}
	defer g.mu.Unlock()
	// fmt.Println(g.Message)
	resp, err := g.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: g.Message,
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
