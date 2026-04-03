package tools

import (
	"github.com/mishrak5j/graphite-mem/internal/config"
	"github.com/mishrak5j/graphite-mem/internal/governor"
	"github.com/mishrak5j/graphite-mem/internal/ingestor"
	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type deps struct {
	ingestor *ingestor.Ingestor
	governor *governor.Governor
	vault    *vault.Vault
	cfg      *config.Config
}

func RegisterAll(
	server *mcp.Server,
	ing *ingestor.Ingestor,
	gov *governor.Governor,
	v *vault.Vault,
	cfg *config.Config,
) {
	d := &deps{
		ingestor: ing,
		governor: gov,
		vault:    v,
		cfg:      cfg,
	}

	registerIngestTool(server, d)
	registerRecallTool(server, d)
	registerForgetTool(server, d)
	registerSuppressTool(server, d)
	registerSessionTool(server, d)
	registerScopeTools(server, d)
	registerStatusTool(server, d)
}
