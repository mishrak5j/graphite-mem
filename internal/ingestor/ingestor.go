package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mishrak5j/graphite-mem/internal/llm"
	"github.com/mishrak5j/graphite-mem/internal/storage"
)

type Ingestor struct {
	llm    llm.Client
	vector storage.VectorStore
	graph  storage.GraphStore
}

func New(llmClient llm.Client, vector storage.VectorStore, graph storage.GraphStore) *Ingestor {
	return &Ingestor{
		llm:    llmClient,
		vector: vector,
		graph:  graph,
	}
}

type IngestResult struct {
	MemoryID        string
	Scope           string
	TriplesExtracted int
}

func (ing *Ingestor) Ingest(ctx context.Context, text, scope string, metadata map[string]any) (*IngestResult, error) {
	memoryID := uuid.New().String()

	triples, err := ing.llm.ExtractTriples(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("extract triples: %w", err)
	}

	embedding, err := ing.llm.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	mem := storage.Memory{
		ID:        memoryID,
		Text:      text,
		Scope:     scope,
		Embedding: embedding,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	errCh := make(chan error, 2)

	go func() {
		errCh <- ing.vector.Add(ctx, mem)
	}()

	go func() {
		storageTriples := llmTriplesToStorage(triples)
		errCh <- ing.graph.MergeTriples(ctx, memoryID, scope, storageTriples)
	}()

	for i := 0; i < 2; i++ {
		if e := <-errCh; e != nil {
			if err == nil {
				err = e
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("store memory: %w", err)
	}

	return &IngestResult{
		MemoryID:        memoryID,
		Scope:           scope,
		TriplesExtracted: len(triples),
	}, nil
}
