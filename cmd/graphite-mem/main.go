package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/mishrak5j/graphite-mem/internal/config"
	"github.com/mishrak5j/graphite-mem/internal/governor"
	"github.com/mishrak5j/graphite-mem/internal/ingestor"
	"github.com/mishrak5j/graphite-mem/internal/llm"
	"github.com/mishrak5j/graphite-mem/internal/storage"
	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/mishrak5j/graphite-mem/resources"
	"github.com/mishrak5j/graphite-mem/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	chromaStore, err := storage.NewChromaStore(cfg.ChromaURL)
	if err != nil {
		log.Fatalf("chroma init: %v", err)
	}

	neo4jStore, err := storage.NewNeo4jStore(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPass)
	if err != nil {
		log.Fatalf("neo4j init: %v", err)
	}
	defer neo4jStore.Close()

	ollamaClient := llm.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel)

	v := vault.New(cfg.DefaultScope)

	ing := ingestor.New(ollamaClient, chromaStore, neo4jStore)

	gov := governor.New(
		ollamaClient,
		chromaStore,
		neo4jStore,
		cfg.DecayLambda,
		cfg.SuppressThreshold,
		cfg.SuppressCooldown,
	)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "graphite-mem", Version: "v0.1.0"},
		nil,
	)

	tools.SetStoreRefs(chromaStore, neo4jStore)
	tools.RegisterAll(server, ing, gov, v, cfg)
	resources.RegisterAll(server, neo4jStore, v)

	switch cfg.Transport {
	case "sse", "http":
		log.Printf("starting graphite-mem MCP server (HTTP/SSE) on %s", cfg.SSEAddr)
		handler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return server },
			nil,
		)
		httpServer := &http.Server{Addr: cfg.SSEAddr, Handler: handler}
		go func() {
			<-ctx.Done()
			httpServer.Close()
		}()
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	default:
		log.Println("starting graphite-mem MCP server (stdio)")
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}
