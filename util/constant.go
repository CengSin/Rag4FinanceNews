package util

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	CollectionName = "financial_articles"
	ModelName      = "openai/gpt-5"
)

var (
	SystemPrompt string
)

func InitSystemPrompt() {
	file, err := os.OpenFile("systemPrompt.md", os.O_RDONLY, 0666)
	if err != nil {
		log.Fatalln("open system prompt file failed, err ", err)
	}

	contextBtys, err := io.ReadAll(file)
	if err != nil {
		log.Fatalln(fmt.Errorf("read system prompt file failed, err %v", err))
	}

	SystemPrompt = string(contextBtys)
}
