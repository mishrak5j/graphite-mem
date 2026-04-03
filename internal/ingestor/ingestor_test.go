package ingestor

import (
	"context"
	"testing"

	"github.com/mishrak5j/graphite-mem/internal/llm"
	"github.com/mishrak5j/graphite-mem/internal/storage"
)

type mockLLM struct {
	triples    []llm.Triple
	embedding  []float32
	tripleErr  error
	embedErr   error
}

func (m *mockLLM) ExtractTriples(_ context.Context, _ string) ([]llm.Triple, error) {
	return m.triples, m.tripleErr
}

func (m *mockLLM) Embed(_ context.Context, _ string) ([]float32, error) {
	return m.embedding, m.embedErr
}

type mockVectorStore struct {
	added   []storage.Memory
	addErr  error
}

func (m *mockVectorStore) Add(_ context.Context, mem storage.Memory) error {
	m.added = append(m.added, mem)
	return m.addErr
}

func (m *mockVectorStore) Query(_ context.Context, _ []float32, _ int, _ storage.ScopeFilter) ([]storage.ScoredMemory, error) {
	return nil, nil
}

func (m *mockVectorStore) Delete(_ context.Context, _ string) error { return nil }
func (m *mockVectorStore) Count(_ context.Context) (int, error)    { return len(m.added), nil }

type mockGraphStore struct {
	triples  []storage.Triple
	mergeErr error
}

func (m *mockGraphStore) MergeTriples(_ context.Context, _ string, _ string, t []storage.Triple) error {
	m.triples = append(m.triples, t...)
	return m.mergeErr
}

func (m *mockGraphStore) QueryByIntent(_ context.Context, _ string, _ storage.ScopeFilter, _ int) ([]storage.RelatedMemory, error) {
	return nil, nil
}

func (m *mockGraphStore) QueryRelated(_ context.Context, _ string, _ storage.ScopeFilter, _ int) ([]storage.RelatedMemory, error) {
	return nil, nil
}

func (m *mockGraphStore) Delete(_ context.Context, _ string) error  { return nil }
func (m *mockGraphStore) NodeCount(_ context.Context) (int, error)  { return 0, nil }
func (m *mockGraphStore) EdgeCount(_ context.Context) (int, error)  { return 0, nil }
func (m *mockGraphStore) Close() error                              { return nil }

func TestIngestSuccess(t *testing.T) {
	ml := &mockLLM{
		triples: []llm.Triple{
			{Subject: "Alice", Predicate: "works_at", Object: "Google"},
		},
		embedding: []float32{0.1, 0.2, 0.3},
	}
	vs := &mockVectorStore{}
	gs := &mockGraphStore{}

	ing := New(ml, vs, gs)

	result, err := ing.Ingest(context.Background(), "Alice works at Google", "/projects/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MemoryID == "" {
		t.Fatal("memory ID should not be empty")
	}
	if result.Scope != "/projects/test" {
		t.Fatalf("expected scope /projects/test, got %s", result.Scope)
	}
	if result.TriplesExtracted != 1 {
		t.Fatalf("expected 1 triple, got %d", result.TriplesExtracted)
	}

	if len(vs.added) != 1 {
		t.Fatalf("expected 1 vector store add, got %d", len(vs.added))
	}
	if vs.added[0].Scope != "/projects/test" {
		t.Fatalf("vector store entry should have scope /projects/test")
	}

	if len(gs.triples) != 1 {
		t.Fatalf("expected 1 graph store triple, got %d", len(gs.triples))
	}
}

func TestIngestNoTriples(t *testing.T) {
	ml := &mockLLM{
		triples:   []llm.Triple{},
		embedding: []float32{0.1, 0.2},
	}
	vs := &mockVectorStore{}
	gs := &mockGraphStore{}

	ing := New(ml, vs, gs)
	result, err := ing.Ingest(context.Background(), "hello", "/default", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TriplesExtracted != 0 {
		t.Fatalf("expected 0 triples, got %d", result.TriplesExtracted)
	}
}

func TestTripleConversion(t *testing.T) {
	input := []llm.Triple{
		{Subject: "A", Predicate: "B", Object: "C"},
		{Subject: "X", Predicate: "Y", Object: "Z"},
	}

	result := llmTriplesToStorage(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 storage triples, got %d", len(result))
	}
	if result[0].Subject != "A" || result[1].Object != "Z" {
		t.Fatal("conversion mismatch")
	}
}
