package tools

import (
	"context"

	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SuppressInput struct {
	Term   string  `json:"term" jsonschema:"Term to suppress (e.g. C++ or Python)"`
	TTL    int     `json:"ttl,omitempty" jsonschema:"Turns before suppression expires (default from config)"`
	Weight float64 `json:"weight,omitempty" jsonschema:"Score multiplier: 0.0 fully hidden, 0.5 halved (default 0.0)"`
}

type SuppressOutput struct {
	Suppressed   string  `json:"suppressed"`
	RemainingTTL int     `json:"remaining_ttl"`
	Weight       float64 `json:"weight"`
}

func registerSuppressTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "suppress_topic",
		Description: "Temporarily suppress a topic from memory recall. Memories containing the suppressed term will have their relevance score reduced for the specified number of turns.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SuppressInput) (*mcp.CallToolResult, SuppressOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		ttl := input.TTL
		if ttl <= 0 {
			ttl = d.cfg.NegWeightTTL
		}

		weight := input.Weight
		if input.TTL == 0 && input.Weight == 0 {
			weight = 0.0
		}

		vault.SuppressTerm(d.vault.Sessions, session.ID, input.Term, ttl, weight)

		return nil, SuppressOutput{
			Suppressed:   input.Term,
			RemainingTTL: ttl,
			Weight:       weight,
		}, nil
	})
}
