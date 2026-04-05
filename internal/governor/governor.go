package governor

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/mishrak5j/graphite-mem/internal/llm"
	"github.com/mishrak5j/graphite-mem/internal/storage"
	"github.com/mishrak5j/graphite-mem/internal/vault"
)

type Governor struct {
	llm        llm.Client
	vector     storage.VectorStore
	graph      storage.GraphStore
	lambda     float64
	suppressor *FrequencySuppressor
}

func New(
	llmClient llm.Client,
	vector storage.VectorStore,
	graph storage.GraphStore,
	lambda float64,
	suppressThreshold, suppressCooldown int,
) *Governor {
	return &Governor{
		llm:        llmClient,
		vector:     vector,
		graph:      graph,
		lambda:     lambda,
		suppressor: NewFrequencySuppressor(suppressThreshold, suppressCooldown),
	}
}

type RecallResult struct {
	Memories  []vault.RankedMemory
	Inhibited bool
}

func (g *Governor) Recall(
	ctx context.Context,
	session *vault.Session,
	query string,
	intent string,
	scopeFilter storage.ScopeFilter,
	suppressions []vault.Suppression,
	topK int,
) (*RecallResult, error) {
	if session.InhibitPastContext {
		return &RecallResult{Inhibited: true}, nil
	}

	embedding, err := g.llm.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	oversample := topK * 2
	if oversample < 10 {
		oversample = 10
	}

	var (
		vectorResults []storage.ScoredMemory
		graphResults  []storage.RelatedMemory
		vectorErr     error
		graphErr      error
		wg            sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		vectorResults, vectorErr = g.vector.Query(ctx, embedding, oversample, scopeFilter)
	}()

	go func() {
		defer wg.Done()
		if intent != "" {
			graphResults, graphErr = g.graph.QueryByIntent(ctx, intent, scopeFilter, oversample)
		} else {
			graphResults, graphErr = g.graph.QueryRelated(ctx, query, scopeFilter, oversample)
		}
	}()

	wg.Wait()

	if vectorErr != nil {
		return nil, vectorErr
	}
	if graphErr != nil {
		return nil, graphErr
	}

	merged := mergeResults(vectorResults, graphResults)

	for i := range merged {
		merged[i].score = applyTemporalDecay(merged[i].score, merged[i].createdAt, g.lambda)
	}

	filtered := make([]candidate, 0, len(merged))
	for _, c := range merged {
		if !g.suppressor.ShouldSuppress(session.ID, c.id, session.TurnCounter) {
			filtered = append(filtered, c)
		}
	}

	ranked := make([]vault.RankedMemory, len(filtered))
	for i, c := range filtered {
		ranked[i] = vault.RankedMemory{
			ID:    c.id,
			Text:  c.text,
			Scope: c.scope,
			Score: c.score,
		}
	}

	ranked = vault.ApplyNegativeWeights(ranked, suppressions)

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	for _, m := range ranked {
		g.suppressor.RecordInjection(session.ID, m.ID, session.TurnCounter)
	}

	return &RecallResult{Memories: ranked}, nil
}

type candidate struct {
	id        string
	text      string
	scope     string
	score     float64
	createdAt time.Time
}

func mergeResults(vectorResults []storage.ScoredMemory, graphResults []storage.RelatedMemory) []candidate {
	byID := make(map[string]*candidate)

	for _, v := range vectorResults {
		byID[v.ID] = &candidate{
			id:        v.ID,
			text:      v.Text,
			scope:     v.Scope,
			score:     v.Score,
			createdAt: v.CreatedAt,
		}
	}

	for _, gr := range graphResults {
		if existing, ok := byID[gr.MemoryID]; ok {
			existing.score *= 1.5
		} else {
			byID[gr.MemoryID] = &candidate{
				id:        gr.MemoryID,
				text:      gr.Text,
				scope:     gr.Scope,
				score:     gr.Score,
				createdAt: gr.CreatedAt,
			}
		}
	}

	result := make([]candidate, 0, len(byID))
	for _, c := range byID {
		result = append(result, *c)
	}
	return result
}
