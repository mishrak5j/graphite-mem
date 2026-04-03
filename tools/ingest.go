package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type IngestInput struct {
	Text     string         `json:"text" jsonschema:"required,description=The text content to store as a memory"`
	Scope    string         `json:"scope,omitempty" jsonschema:"description=Scope path (e.g. /projects/my-app). Uses session default if omitted"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"description=Optional metadata (intent, source, tags)"`
}

type IngestOutput struct {
	MemoryID         string `json:"memory_id"`
	Scope            string `json:"scope"`
	TriplesExtracted int    `json:"triples_extracted"`
}

func registerIngestTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ingest_memory",
		Description: "Store a new memory with automatic triple extraction and embedding. Tags memory with a scope for multi-tenant retrieval.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input IngestInput) (*mcp.CallToolResult, IngestOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		scope := input.Scope
		if scope == "" {
			if len(session.MountedScopes) > 0 {
				scope = session.MountedScopes[0]
			} else {
				scope = d.cfg.DefaultScope
			}
		}

		d.vault.Scopes.CreateScope(scope, scope)

		result, err := d.ingestor.Ingest(ctx, input.Text, scope, input.Metadata)
		if err != nil {
			return errorResult(fmt.Sprintf("ingest failed: %v", err)), IngestOutput{}, nil
		}

		d.vault.Scopes.IncrementMemoryCount(scope)

		return nil, IngestOutput{
			MemoryID:         result.MemoryID,
			Scope:            result.Scope,
			TriplesExtracted: result.TriplesExtracted,
		}, nil
	})
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func jsonText(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}
