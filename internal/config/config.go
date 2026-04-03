package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Transport
	Transport string // "stdio" or "sse"
	SSEAddr   string

	// Storage
	ChromaURL  string
	Neo4jURI   string
	Neo4jUser  string
	Neo4jPass  string

	// LLM
	OllamaURL   string
	OllamaModel string

	// Governor
	DecayLambda       float64
	SuppressThreshold int
	SuppressCooldown  int

	// Vault
	DefaultScope      string
	NegWeightTTL      int
}

func Load() *Config {
	return &Config{
		Transport:         envOrDefault("GRAPHITE_TRANSPORT", "stdio"),
		SSEAddr:           envOrDefault("GRAPHITE_SSE_ADDR", ":3100"),
		ChromaURL:         envOrDefault("GRAPHITE_CHROMA_URL", "http://localhost:8000"),
		Neo4jURI:          envOrDefault("GRAPHITE_NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:         envOrDefault("GRAPHITE_NEO4J_USER", "neo4j"),
		Neo4jPass:         envOrDefault("GRAPHITE_NEO4J_PASS", "graphite"),
		OllamaURL:         envOrDefault("GRAPHITE_OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:       envOrDefault("GRAPHITE_OLLAMA_MODEL", "llama3.1"),
		DecayLambda:       envOrDefaultFloat("GRAPHITE_DECAY_LAMBDA", 0.01),
		SuppressThreshold: envOrDefaultInt("GRAPHITE_SUPPRESS_THRESHOLD", 3),
		SuppressCooldown:  envOrDefaultInt("GRAPHITE_SUPPRESS_COOLDOWN", 5),
		DefaultScope:      envOrDefault("GRAPHITE_DEFAULT_SCOPE", "/default"),
		NegWeightTTL:      envOrDefaultInt("GRAPHITE_NEG_WEIGHT_DEFAULT_TTL", 50),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
