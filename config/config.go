package config

type QdrantConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"baseURL"`
	ApiKey  string `yaml:"apiKey"`
}

type TemporalConfig struct {
	HostPort string `yaml:"hostPort"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"DB"`
}

type Config struct {
	Qdrant    *QdrantConfig   `yaml:"qdrant"`
	OpenAI    *OpenAIConfig   `yaml:"openAI"`
	Temporal  *TemporalConfig `yaml:"temporal"`
	McpServer string          `yaml:"mcpServer"`
	Redis     *RedisConfig    `yaml:"redis"`
}
