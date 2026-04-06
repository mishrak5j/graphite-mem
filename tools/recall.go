package tools

import (
	"context"
	"fmt"

	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RecallInput struct {
	Query      string `json:"query" jsonschema:"Search query for memory recall"`
	Intent     string `json:"intent,omitempty" jsonschema:"Intent filter for graph-based retrieval"`
	TopK       int    `json:"top_k,omitempty" jsonschema:"Max memories to return (default 5)"`
	CrossScope bool   `json:"cross_scope,omitempty" jsonschema:"Search all scopes regardless of mount state"`
}

type RecallOutput struct {
	Memories  []MemoryItem `json:"memories"`
	SessionID string       `json:"session_id"`
	Inhibited bool         `json:"inhibited"`
}

type MemoryItem struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
	Scope string  `json:"scope"`
	ID    string  `json:"id"`
}

func registerRecallTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall_memory",
		Description: "Retrieve relevant memories using hybrid graph-vector search with temporal decay and scope filtering. Supports cross-scope deep references.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		topK := input.TopK
		if topK <= 0 {
			topK = 5
		}

		scopeFilter := d.vault.Scopes.ResolveScopeFilter(session, input.CrossScope)
		suppressions := vault.GetActiveSuppressions(d.vault.Sessions, session.ID)

		result, err := d.governor.Recall(ctx, session, input.Query, input.Intent, scopeFilter, suppressions, topK)
		if err != nil {
			return errorResult(fmt.Sprintf("recall failed: %v", err)), RecallOutput{}, nil
		}

		d.vault.Sessions.IncrementTurn(session.ID)

		items := make([]MemoryItem, len(result.Memories))
		for i, m := range result.Memories {
			items[i] = MemoryItem{
				Text:  m.Text,
				Score: m.Score,
				Scope: m.Scope,
				ID:    m.ID,
			}
		}

		return nil, RecallOutput{
			Memories:  items,
			SessionID: session.ID,
			Inhibited: result.Inhibited,
		}, nil
	})
}
