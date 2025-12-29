package main

import (
	"context"
	"github.com/qdrant/go-client/qdrant"
	"log"
)

// 配置常量
const (
	CollectionName = "fupengshuo_articles"
	VectorSize     = 1536 // 对应 qwen/qwen3-embedding-8b
	QdrantHost     = "localhost"
	QdrantPort     = 6334
)

func main() {

	// 1. 连接 Qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: QdrantHost,
		Port: QdrantPort,
	})
	if err != nil {
		log.Fatalln("qdrant connection failed, err ", err.Error())
	}
	defer client.Close()

	_ = EnsureFullTextIndex(context.Background(), client)
}
