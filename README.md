# Graphite Memory

**Hybrid graph + vector memory for LLMs, exposed as an MCP server.**

Memory that behaves like memory—not a flat vector dump. Temporal decay, scoped tenants, graph–vector fusion, and MCP-native tools.

**Stack:** Go · [Model Context Protocol](https://modelcontextprotocol.io) · Neo4j 5 · Chroma · [Ollama](https://ollama.ai) · MIT License

---

## Contents

- [The problem](#the-problem)
- [What Graphite does](#what-graphite-does)
- [Architecture](#architecture)
- [Quick start](#quick-start)
- [Claude Desktop](#claude-desktop-stdio-mcp)
- [Cursor](#cursor)
- [Data flow](#data-flow)
- [MCP tools](#mcp-tool-reference)
- [MCP resources](#mcp-resources)
- [Configuration](#configuration)
- [Project layout](#project-layout)
- [Design notes](#design-notes)
- [Makefile](#makefile)

---

## The problem

Most LLM “memory” is flat retrieval: everything competes on similarity, important context repeats like a broken record, and the system has no notion of *why* a fact mattered—only that it was similar to the query.

## What Graphite does

Graphite Memory is a **Model Context Protocol server** in Go. It combines vector search with a property graph so recall can rank by both relevance and structure.

| Capability | What it solves |
|:---|:---|
| **Graph + vector fusion** | Chroma finds *what* is similar; Neo4j carries *how* ideas connect. Items that hit both stores get a **1.5×** score boost. |
| **Temporal decay** | Newer memories rank higher. Score is scaled by `e^(−λΔt)` with Δt in **hours**; default λ is `0.01`, or set `GRAPHITE_DECAY_HALF_LIFE_DAYS` for half-life in days. |
| **Frequency suppression** | After a memory is injected N times in a session, it backs off for K turns so the same line does not dominate. |
| **Scoped memory** | Paths like `/projects/architect-cli` isolate context; mount and unmount scopes per session. |
| **Context virtualization** | Start an inhibited session to block past memories without deleting them. |
| **Negative weighting** | Temporarily down-rank topics (e.g. “not C++ right now”) with TTL decay. |

---

## Architecture

```
                    ┌─────────────────────────┐
                    │      LLM  clients       │
                    │Gemini · ChatGPT · Claude│
                    └───────────┬─────────────┘
                                │
                          MCP (stdio / HTTP)
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                                                                  │
│   ┌────────────────────────────────────────────────────────────┐ │
│   │                      MCP  tools                            │ │
│   │                                                            │ │
│   │  ingest_memory    recall_memory     forget_memory          │ │
│   │  suppress_topic   new_session       memory_status          │ │
│   │  mount_scope      unmount_scope     list_scopes            │ │
│   └──────────────┬──────────────────────┬──────────────────────┘ │
│                  │                      │                        │
│   ┌──────────────▼──────┐  ┌────────────▼───────────────┐        │
│   │    Memory  vault    │  │        Governor            │        │
│   │                     │  │                            │        │
│   │  Session manager    │  │  Parallel retrieval (WG)   │        │
│   │  Scope registry     │  │  Temporal decay  e^(−λt)   │        │
│   │  Inhibit toggle     │  │  Frequency suppressor      │        │
│   │  Negative weights   │  │  Graph boost  (×1.5)       │        │
│   └─────────────────────┘  └──────┬──────────┬──────────┘        │
│                                   │          │                   │
│                          ┌────────▼──┐  ┌────▼────────┐          │
│                          │ Chroma    │  │   Neo4j     │          │
│                          │ (vectors) │  │   (graph)   │          │
│                          └───────────┘  └─────────────┘          │
│                                                                  │
│   ┌────────────────────────────────────────────────────────────┐ │
│   │         Ollama (e.g. Llama 3.1)                            │ │
│   │         Triple extraction · embedding generation           │ │
│   └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│                        Graphite Memory server                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## Quick start

### Prerequisites

- **Go 1.26+** (see `go.mod`)
- **Docker** and **Docker Compose** (Neo4j + Chroma)
- **[Ollama](https://ollama.ai)** with `llama3.1` (or another model you configure)

### 1. Clone and start data stores

Set a Neo4j password (no default in the repo). Copy [`.env.example`](.env.example) to `.env` in the repo root and set `GRAPHITE_NEO4J_PASS` to a strong value. Docker Compose and `graphite-mem` both use this variable.

```bash
git clone https://github.com/mishrak5j/graphite-mem.git
cd graphite-mem

cp .env.example .env
# Edit .env and set GRAPHITE_NEO4J_PASS

make docker-up
ollama pull llama3.1
```

Docker Compose loads `.env` from the repo root when you run `make docker-up` from this directory.

Neo4j nodes are keyed by `(scope, name)`. If you are upgrading from an older graph that used only global names, run `make docker-reset` once to recreate the Neo4j volume.

### 2. Build and run

From the repo root, `graphite-mem` loads a `.env` file if present (same file as Docker Compose). Alternatively, export `GRAPHITE_NEO4J_PASS` in your shell.

```bash
make build

# Default: stdio (for MCP clients that spawn the process)
./bin/graphite-mem

# Optional: HTTP/SSE on port 3100
GRAPHITE_TRANSPORT=sse GRAPHITE_SSE_ADDR=:3100 ./bin/graphite-mem
```

### 3. Verify

```bash
make test        # go test ./... -v -race
make lint        # go vet
```

---

## Claude Desktop (stdio MCP)

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS (paths differ on Windows/Linux) and add something like:

```json
{
  "mcpServers": {
    "graphite-mem": {
      "command": "/absolute/path/to/graphite-mem/bin/graphite-mem",
      "cwd": "/absolute/path/to/graphite-mem"
    }
  }
}
```

Restart Claude Desktop. Keep `make docker-up` and Ollama running while you use the tools.

---

## Cursor

Add an MCP entry that runs the built binary (example: [`.cursor/mcp.json`](.cursor/mcp.json) with `command` set to `${workspaceFolder}/bin/graphite-mem`).

1. `make build`
2. `make docker-up` and ensure your Ollama model is available (`ollama pull llama3.1` or your `GRAPHITE_OLLAMA_MODEL`)
3. Reload MCP in Cursor (Command Palette → “MCP: Restart”, or restart the editor)

If `${workspaceFolder}` is not expanded in your environment, set `command` to the absolute path of `bin/graphite-mem`.

---

## Data flow

### Ingest

```
text + scope
     │
     ▼
┌──────────┐     ┌──────────────────────┐
│  Ollama  │────▶│  Extract triples     │──── (subject, predicate, object)
│          │     │  Generate embedding  │──── float32 vector
└──────────┘     └──────────────────────┘
                          │
               ┌──────────┴──────────┐
               ▼                     ▼
        ┌─────────────┐      ┌─────────────┐
        │  Chroma     │      │   Neo4j     │
        │  store doc  │      │  MERGE rels │
        └─────────────┘      └─────────────┘
            (parallel — two goroutines)
```

### Recall

```
query + intent + scope filter
     │
     ▼
  Embed query
     │
     ├──── goroutine ──▶  Chroma vector search (oversampled 2×)
     │
     ├──── goroutine ──▶  Neo4j traversal (intent / related)
     │
     ▼  (sync.WaitGroup)
  Merge ──▶ graph boost (1.5×) on dual hits
     │
     ▼
  Temporal decay ──▶ score × e^(−λ × hours)
     │
     ▼
  Frequency suppression
     │
     ▼
  Negative weights ──▶ drop or down-rank when weight ≤ threshold
     │
     ▼
  Sort · trim to top_k · record injection counts
```

---

## MCP tool reference

### `ingest_memory`

Store a memory. Triples are extracted; an embedding is stored in Chroma and relationships in Neo4j.

```json
{
  "text": "I'm building a CLI tool in Go called Architect",
  "scope": "/projects/architect-cli",
  "metadata": {}
}
```

Returns `memory_id`, `scope`, `triples_extracted`.

### `recall_memory`

Hybrid retrieval with ranking stages above.

```json
{
  "query": "CLI tools I've been working on",
  "intent": "find active projects",
  "top_k": 5,
  "cross_scope": false
}
```

Returns ranked memories, or `inhibited: true` if the session blocks past context.

### `new_session`

Start a new session; optionally inhibit prior memories or pre-mount scopes.

```json
{ "inhibit": true, "mount_scopes": ["/projects/architect-cli"] }
```

### `suppress_topic`

Temporarily suppress a topic from recall (TTL decay per turn).

```json
{ "term": "C++", "ttl": 50, "weight": 0.0 }
```

### `forget_memory`

Delete a memory from vector and graph stores.

```json
{ "memory_id": "uuid-here" }
```

### `mount_scope` / `unmount_scope`

```json
{ "scope_path": "/projects/architect-cli" }
```

### `list_scopes`

Lists scopes with mount status and counts.

### `memory_status`

Session state, mounts, suppressions, and store stats.

---

## MCP resources

| URI | Description |
|:----|:------------|
| `memory://profile` | Plain-text profile: session id, mounted scopes, graph-derived triples. |

---

## Configuration

Environment variables use the `GRAPHITE_` prefix.

<details>
<summary><strong>Full variable reference</strong></summary>

| Variable | Default | Description |
|:---|:---|:---|
| `GRAPHITE_TRANSPORT` | `stdio` | `stdio` or `sse` |
| `GRAPHITE_SSE_ADDR` | `:3100` | HTTP listen address in SSE mode |
| `GRAPHITE_CHROMA_URL` | `http://localhost:8000` | Chroma HTTP API |
| `GRAPHITE_NEO4J_URI` | `bolt://localhost:7687` | Neo4j Bolt URI |
| `GRAPHITE_NEO4J_USER` | `neo4j` | Neo4j user |
| `GRAPHITE_NEO4J_PASS` | *(required)* | Neo4j password; no default. Use the same value in `.env` for Docker Compose. |
| `GRAPHITE_OLLAMA_URL` | `http://localhost:11434` | Ollama base URL |
| `GRAPHITE_OLLAMA_MODEL` | `llama3.1` | Model for triples + embeddings |
| `GRAPHITE_DECAY_LAMBDA` | `0.01` | λ per hour: `score × e^(−λ·hours)`. `0` disables. Ignored if half-life is set. |
| `GRAPHITE_DECAY_HALF_LIFE_DAYS` | *(unset)* | If set to a positive number, λ = `ln(2) / (days × 24)`. |
| `GRAPHITE_SUPPRESS_THRESHOLD` | `3` | Injections before frequency cooldown |
| `GRAPHITE_SUPPRESS_COOLDOWN` | `5` | Turns to hide an over-injected memory |
| `GRAPHITE_DEFAULT_SCOPE` | `/default` | Default scope path |
| `GRAPHITE_NEG_WEIGHT_DEFAULT_TTL` | `50` | Default suppression TTL (turns) |

</details>

---

## Project layout

```
graphite-mem/
├── .env.example            # Template for `GRAPHITE_NEO4J_PASS` (copy to `.env`)
├── cmd/graphite-mem/
│   └── main.go              # Transport, wiring, server bootstrap
├── internal/
│   ├── config/              # Environment config
│   ├── llm/                 # Ollama: triples + embeddings
│   ├── ingestor/            # Parallel Chroma + Neo4j ingest
│   ├── governor/            # Hybrid recall + ranking
│   ├── vault/               # Sessions, scopes, suppression, inhibit
│   └── storage/             # Chroma + Neo4j clients
├── tools/                   # MCP tool definitions
├── resources/               # MCP resource: memory://profile
├── scripts/
│   ├── docker-compose.yml   # Neo4j + Chroma
│   └── seed.cypher          # Optional seed
├── Makefile
└── README.md
```

---

## Design notes

| Topic | Rationale |
|:---|:---|
| **Go** | Goroutines fit parallel vector + graph queries; keeps latency predictable for MCP clients. |
| **Graph + vector** | Vectors approximate *similarity*; the graph captures *relationships* and intent chains. Together they read more like structured recall than keyword RAG. |
| **Scopes** | Path-based scopes avoid cross-project bleed without re-indexing. |
| **Frequency suppression** | High-similarity chunks should not dominate every turn; cooldown brings them back after a pause. |
| **Negative weights** | Short-lived “don’t mention X” without deleting stored memories. |

---

## Makefile

```bash
make build          # → bin/graphite-mem
make run            # stdio
make run-sse        # SSE on :3100
make test           # go test ./... -v -race
make lint           # go vet ./...
make docker-up      # Neo4j + Chroma
make docker-down    # stop containers
make docker-reset   # remove volumes and restart
make tidy           # go mod tidy
make clean          # rm -rf bin/
```

---

MIT License
