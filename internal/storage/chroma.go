package storage

import (
	"context"
	"fmt"
	"time"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const collectionName = "graphite_memories"

type ChromaStore struct {
	client     chroma.Client
	collection chroma.Collection
}

func NewChromaStore(url string) (*ChromaStore, error) {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(url))
	if err != nil {
		return nil, fmt.Errorf("chroma client: %w", err)
	}

	col, err := client.GetOrCreateCollection(context.Background(), collectionName)
	if err != nil {
		return nil, fmt.Errorf("chroma collection: %w", err)
	}

	return &ChromaStore{client: client, collection: col}, nil
}

func (s *ChromaStore) Add(ctx context.Context, mem Memory) error {
	meta, err := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
		"scope":      mem.Scope,
		"created_at": mem.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("build metadata: %w", err)
	}

	emb := &embeddings.Float32Embedding{}
	if err := emb.FromFloat32(mem.Embedding...); err != nil {
		return fmt.Errorf("build embedding: %w", err)
	}

	return s.collection.Add(ctx,
		chroma.WithIDs(chroma.DocumentID(mem.ID)),
		chroma.WithTexts(mem.Text),
		chroma.WithMetadatas(meta),
		chroma.WithEmbeddings(emb),
	)
}

func (s *ChromaStore) Query(ctx context.Context, embedding []float32, topK int, filter ScopeFilter) ([]ScoredMemory, error) {
	emb := &embeddings.Float32Embedding{}
	if err := emb.FromFloat32(embedding...); err != nil {
		return nil, fmt.Errorf("build query embedding: %w", err)
	}

	opts := []chroma.CollectionQueryOption{
		chroma.WithQueryEmbeddings(emb),
		chroma.WithNResults(topK),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas, chroma.IncludeDistances),
	}

	if !filter.CrossScope && len(filter.Scopes) > 0 {
		opts = append(opts, chroma.WithWhere(
			chroma.InString("scope", filter.Scopes...),
		))
	}

	qr, err := s.collection.Query(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("chroma query: %w", err)
	}

	idGroups := qr.GetIDGroups()
	docGroups := qr.GetDocumentsGroups()
	distGroups := qr.GetDistancesGroups()
	metaGroups := qr.GetMetadatasGroups()

	if len(idGroups) == 0 || len(idGroups[0]) == 0 {
		return nil, nil
	}

	ids := idGroups[0]
	var results []ScoredMemory

	for i, id := range ids {
		mem := ScoredMemory{
			Memory: Memory{
				ID: string(id),
			},
		}

		if len(docGroups) > 0 && i < len(docGroups[0]) && docGroups[0][i] != nil {
			mem.Text = docGroups[0][i].ContentString()
		}

		if len(distGroups) > 0 && i < len(distGroups[0]) {
			mem.Score = 1.0 - float64(distGroups[0][i])
		}

		if len(metaGroups) > 0 && i < len(metaGroups[0]) && metaGroups[0][i] != nil {
			if scope, ok := metaGroups[0][i].GetString("scope"); ok {
				mem.Scope = scope
			}
			if ts, ok := metaGroups[0][i].GetString("created_at"); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					mem.CreatedAt = t
				}
			}
		}

		results = append(results, mem)
	}

	return results, nil
}

func (s *ChromaStore) Delete(ctx context.Context, id string) error {
	return s.collection.Delete(ctx, chroma.WithIDs(chroma.DocumentID(id)))
}

func (s *ChromaStore) Count(ctx context.Context) (int, error) {
	return s.collection.Count(ctx)
}
