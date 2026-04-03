package storage

import (
	"context"
	"time"
)

type Memory struct {
	ID        string
	Text      string
	Scope     string
	Embedding []float32
	Metadata  map[string]any
	CreatedAt time.Time
}

type ScopeFilter struct {
	Scopes     []string
	CrossScope bool
}

type ScoredMemory struct {
	Memory
	Score float64
}

type Triple struct {
	Subject   string
	Predicate string
	Object    string
}

type RelatedMemory struct {
	MemoryID string
	Text     string
	Scope    string
	Path     string
	Score    float64
}

type VectorStore interface {
	Add(ctx context.Context, mem Memory) error
	Query(ctx context.Context, embedding []float32, topK int, filter ScopeFilter) ([]ScoredMemory, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

type GraphStore interface {
	MergeTriples(ctx context.Context, memoryID string, scope string, triples []Triple) error
	QueryByIntent(ctx context.Context, intent string, filter ScopeFilter, limit int) ([]RelatedMemory, error)
	QueryRelated(ctx context.Context, subject string, filter ScopeFilter, limit int) ([]RelatedMemory, error)
	Delete(ctx context.Context, memoryID string) error
	NodeCount(ctx context.Context) (int, error)
	EdgeCount(ctx context.Context) (int, error)
	Close() error
}
