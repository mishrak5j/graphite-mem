package llm

import "context"

type Triple struct {
	Subject   string `json:"s"`
	Predicate string `json:"p"`
	Object    string `json:"o"`
}

type Client interface {
	ExtractTriples(ctx context.Context, text string) ([]Triple, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}
