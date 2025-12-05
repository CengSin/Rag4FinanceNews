package client

import (
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/client"
	"log"
	"os"
	"rag4financenew/config"
)

var (
	Qdrant   *qdrant.Client
	AI       *openai.Client
	Temporal client.Client
	Redis    *redis.Client
)

func InitQdrant(cfg *config.QdrantConfig) {
	if cfg == nil {
		panic("qdrant config is nil")
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Host,
		Port: cfg.Port,
	})
	if err != nil {
		log.Fatalln("qdrant client init failed, err ", err)
	}

	Qdrant = client
}

func InitLLMs(cfg *config.OpenAIConfig) {
	if cfg == nil {
		panic("openai config is nil")
	}
	if cfg.ApiKey == "" {
		cfg.ApiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	conf := openai.DefaultConfig(cfg.ApiKey)
	if len(cfg.BaseURL) > 0 {
		conf.BaseURL = cfg.BaseURL
	}
	AI = openai.NewClientWithConfig(conf)
}

func InitTemporal(cfg *config.TemporalConfig) {
	if cfg == nil {
		panic("temporal config is nil")
	}

	cl, err := client.Dial(client.Options{
		HostPort: cfg.HostPort,
	})
	if err != nil {
		log.Fatalln("connntect to temporal failed, err ", err)
	}
	Temporal = cl
}

func InitRedis(cfg *config.RedisConfig) {
	if cfg == nil {
		panic("temporal config is nil")
	}

	Redis = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password, // no password set
		DB:       cfg.DB,       // use default DB
	})
}

func Close() {
	Temporal.Close()
	Qdrant.Close()
	McpClient.Close()
	Redis.Close()
}
