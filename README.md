# Graphite-Memory

A high-performance cognitive middleware for LLM personalization, built in Go. Connects to Gemini and ChatGPT via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io).

## What It Does

Standard LLM memory (RAG) is flat -- it treats every past interaction with equal importance and often gets stuck in semantic loops. Graphite-Memory solves this with:

- **Hybrid Graph-Vector Architecture** -- ChromaDB for *what* you said, Neo4j for *why* you said it
- **Temporal Decay** -- recent memories outrank stale ones via `Score = Similarity × e^(-λΔt)`
- **Frequency Suppression** -- prevents the "broken record" problem by cooling down overused facts
- **Scoped Multi-Tenant Memory** -- memories siloed into scopes (e.g., `/projects/architect-cli`, `/personal/learning`)
- **Context Virtualization** -- "private browsing" mode for AI that blocks all past memories without deleting them
- **Negative Memory Weighting** -- temporarily suppress topics ("forget the C++ stuff for now")

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    LLM Clients                       │
│              (Gemini / ChatGPT / Claude)             │
└─────────────────┬────────────────────────────────────┘
                  │ MCP (stdio / HTTP)
┌─────────────────▼────────────────────────────────────┐
│              MCP Server (Go)                         │
│  ┌─────────────────────────────────────────────────┐ │
│  │            10 MCP Tools                         │ │
│  │  ingest_memory · recall_memory · forget_memory  │ │
│  │  suppress_topic · new_session                   │ │
│  │  mount_scope · unmount_scope · list_scopes      │ │
│  │  memory_status                                  │ │
│  └──────────┬──────────────────────────────────────┘ │
│  ┌──────────▼──────────────────────────────────────┐ │
│  │          Memory Vault                           │ │
│  │  Session Manager · Scope Registry               │ │
│  │  Negative Weighting · Inhibit Toggle            │ │
│  └──────────┬──────────────────────────────────────┘ │
│  ┌──────────▼──────────────────────────────────────┐ │
│  │          Governor (Core Engine)                  │ │
│  │  Parallel Retrieval (goroutines)                │ │
│  │  Temporal Decay · Frequency Suppression         │ │
│  └──────┬───────────────────┬──────────────────────┘ │
│         │                   │                        │
│  ┌──────▼──────┐    ┌──────▼──────┐                  │
│  │  ChromaDB   │    │   Neo4j     │                  │
│  │  (Vectors)  │    │   (Graph)   │                  │
│  └─────────────┘    └─────────────┘                  │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │  Ollama (Llama 3.1) -- Triple Extraction + Embed│ │
│  └─────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- [Ollama](https://ollama.ai) with `llama3.1` model

### Setup

```bash
# Clone
git clone https://github.com/mishrak5j/graphite-mem.git
cd graphite-mem

# Start Neo4j + ChromaDB
make docker-up

# Pull the Ollama model
ollama pull llama3.1

# Build and run (stdio mode)
make build
./bin/graphite-mem

# Or run in HTTP mode
GRAPHITE_TRANSPORT=sse GRAPHITE_SSE_ADDR=:3100 ./bin/graphite-mem
```

### Run Tests

```bash
make test
```

## MCP Tools

### `ingest_memory`
Store a memory with automatic triple extraction and scope tagging.
```json
{"text": "I'm building a CLI tool in Go called Architect", "scope": "/projects/architect-cli"}
```

### `recall_memory`
Retrieve relevant memories with hybrid search, temporal decay, and scope filtering.
```json
{"query": "CLI tools", "top_k": 5, "cross_scope": false}
```

### `new_session`
Create a fresh session. Use `inhibit: true` for clean-slate mode.
```json
{"inhibit": true}
```

### `mount_scope` / `unmount_scope`
Control which memory scopes are visible in the current session.
```json
{"scope_path": "/projects/architect-cli"}
```

### `suppress_topic`
Temporarily suppress a topic from recall results.
```json
{"term": "C++", "ttl": 50, "weight": 0.0}
```

### `forget_memory`
Permanently delete a memory from both stores.
```json
{"memory_id": "uuid-here"}
```

### `list_scopes`
List all scopes with mount status and memory counts.

### `memory_status`
Full diagnostic: session state, mounted scopes, active suppressions, store stats.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `GRAPHITE_TRANSPORT` | `stdio` | Transport mode: `stdio` or `sse` |
| `GRAPHITE_SSE_ADDR` | `:3100` | HTTP listen address |
| `GRAPHITE_CHROMA_URL` | `http://localhost:8000` | ChromaDB endpoint |
| `GRAPHITE_NEO4J_URI` | `bolt://localhost:7687` | Neo4j bolt URI |
| `GRAPHITE_NEO4J_USER` | `neo4j` | Neo4j username |
| `GRAPHITE_NEO4J_PASS` | `graphite` | Neo4j password |
| `GRAPHITE_OLLAMA_URL` | `http://localhost:11434` | Ollama endpoint |
| `GRAPHITE_OLLAMA_MODEL` | `llama3.1` | Model for triple extraction and embedding |
| `GRAPHITE_DECAY_LAMBDA` | `0.01` | Temporal decay rate (~3 day half-life) |
| `GRAPHITE_SUPPRESS_THRESHOLD` | `3` | Inject count before frequency cooldown |
| `GRAPHITE_SUPPRESS_COOLDOWN` | `5` | Turns to hide a frequently injected fact |
| `GRAPHITE_DEFAULT_SCOPE` | `/default` | Default memory scope |
| `GRAPHITE_NEG_WEIGHT_DEFAULT_TTL` | `50` | Default suppression TTL in turns |

## Project Structure

```
graphite-mem/
├── cmd/graphite-mem/main.go       # Entry point, wiring, transport selection
├── internal/
│   ├── vault/                     # Session management, scopes, negative weighting
│   ├── governor/                  # Parallel retrieval, decay, frequency suppression
│   ├── ingestor/                  # Triple extraction + dual-store ingest pipeline
│   ├── storage/                   # ChromaDB + Neo4j drivers with scope filtering
│   ├── llm/                       # Ollama HTTP client
│   └── config/                    # Environment-based configuration
├── tools/                         # 10 MCP tool definitions
├── resources/                     # MCP resource: memory://profile
├── scripts/                       # Docker Compose for Neo4j + ChromaDB
├── Makefile
└── README.md
```

## Key Technical Decisions

**Why Go?** Goroutines query vector and graph databases in parallel, merging results in <15ms. No GC pauses keeps time-to-first-token low.

**Why Hybrid Graph+Vector?** Vector search finds *what* is similar. Graph traversal finds *why* it matters (intent, goals, relationships).

**Why Scoped Memory?** Multi-tenant scoping prevents context pollution between projects and enables instant "context swap" by mounting different scopes.

## License

MIT
