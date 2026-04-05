<div align="center">

<a href="https://github.com/mishrak5j/graphite-mem">
  <img src="resources/readme-banner.svg" alt="Graphite Memory — hybrid graph and vector memory for LLMs via MCP" width="100%" />
</a>

**Memory that behaves like memory — not a flat vector dump.**

<sub>Temporal decay · Scoped tenants · Graph + vector fusion · MCP-native tools</sub>

<br>

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-Server-8B5CF6?style=for-the-badge)](https://modelcontextprotocol.io)
[![Neo4j](https://img.shields.io/badge/Neo4j-5-4581C3?style=for-the-badge&logo=neo4j&logoColor=white)](https://neo4j.com)
[![Chroma](https://img.shields.io/badge/Chroma-DB-FF6F61?style=for-the-badge)](https://www.trychroma.com)
[![License](https://img.shields.io/badge/License-MIT-22c55e?style=for-the-badge)](LICENSE)

<br>

---

</div>

## The Problem

Standard LLM memory is flat. Every RAG system treats past interactions with equal importance, returns the same facts over and over, and has zero awareness of *why* something was said — only *what* was said.

Your AI remembers everything. It understands nothing.

## The Fix

Graphite-Memory is a **Model Context Protocol server** written in Go that replaces naive retrieval with a hybrid cognitive layer:

| Capability | What it solves |
|:---|:---|
| **Graph + Vector Fusion** | ChromaDB finds *what* is similar. Neo4j finds *why* it matters — intent, goals, relationships between ideas. Dual hits get a **1.5× score boost**. |
| **Temporal Decay** | Recent memories outrank stale ones. Score decays as `e^(−λΔt)` with Δt in **hours**; default λ is `0.01`, or set `GRAPHITE_DECAY_HALF_LIFE_DAYS` instead. |
| **Frequency Suppression** | Stops the "broken record" problem. After a memory is injected N times in a session, it cools down for K turns. |
| **Scoped Memory** | Memories are siloed into paths — `/projects/architect-cli`, `/personal/learning`. Mount and unmount scopes per session. Zero cross-contamination. |
| **Context Virtualization** | "Private browsing" for AI. Start an inhibited session — all past memories are blocked, nothing is deleted. |
| **Negative Weighting** | Temporarily suppress topics: *"forget the C++ stuff for now"*. Suppressed terms decay per-turn via TTL. |

<br>

## Architecture

```
                    ┌─────────────────────────┐
                    │      LLM  Clients       │
                    │  Gemini · ChatGPT · Claude│
                    └───────────┬─────────────┘
                                │
                          MCP (stdio / HTTP)
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                                                                  │
│   ┌────────────────────────────────────────────────────────────┐ │
│   │                      MCP  TOOLS                            │ │
│   │                                                            │ │
│   │  ingest_memory    recall_memory     forget_memory          │ │
│   │  suppress_topic   new_session       memory_status          │ │
│   │  mount_scope      unmount_scope     list_scopes            │ │
│   └──────────────┬──────────────────────┬──────────────────────┘ │
│                  │                      │                        │
│   ┌──────────────▼──────┐  ┌────────────▼───────────────┐       │
│   │    MEMORY  VAULT    │  │        GOVERNOR             │       │
│   │                     │  │                             │       │
│   │  Session Manager    │  │  Parallel Retrieval (WG)    │       │
│   │  Scope Registry     │  │  Temporal Decay  e^(−λt)    │       │
│   │  Inhibit Toggle     │  │  Frequency Suppressor       │       │
│   │  Negative Weights   │  │  Graph Boost  (×1.5)        │       │
│   └─────────────────────┘  └──────┬──────────┬──────────┘       │
│                                   │          │                   │
│                          ┌────────▼──┐  ┌────▼────────┐         │
│                          │ ChromaDB  │  │   Neo4j     │         │
│                          │ (Vectors) │  │   (Graph)   │         │
│                          └───────────┘  └─────────────┘         │
│                                                                  │
│   ┌────────────────────────────────────────────────────────────┐ │
│   │         Ollama  (Llama 3.1)                                │ │
│   │         Triple Extraction  ·  Embedding Generation         │ │
│   └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│                        GRAPHITE-MEMORY  SERVER                   │
└──────────────────────────────────────────────────────────────────┘
```

<br>

## Quick Start

### Prerequisites

- **Go 1.26+** (see `go.mod`)
- **Docker & Docker Compose** (for Neo4j + ChromaDB)
- **[Ollama](https://ollama.ai)** with `llama3.1` pulled

### 1. Clone & Boot Infrastructure

```bash
git clone https://github.com/mishrak5j/graphite-mem.git
cd graphite-mem

# Spin up Neo4j + ChromaDB
make docker-up

# Pull the embedding / extraction model
ollama pull llama3.1
```

Neo4j entities are keyed by `(scope, name)`. If you are upgrading from an older graph that only used global names, run `make docker-reset` once to recreate the Neo4j volume.

### 2. Build & Run

```bash
# Build the binary
make build

# Run in stdio mode (default — pipe from your MCP client)
./bin/graphite-mem

# Or run in HTTP/SSE mode on port 3100
GRAPHITE_TRANSPORT=sse GRAPHITE_SSE_ADDR=:3100 ./bin/graphite-mem
```

### 3. Verify

```bash
make test        # go test ./... -v -race
make lint        # go vet
```

### 4. Claude Desktop (stdio MCP)

Point Claude Desktop at the built binary so it can spawn the server. On macOS, edit `~/Library/Application Support/Claude/claude_desktop_config.json` and merge something like:

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

Use your real paths. Restart Claude Desktop. Keep `make docker-up` (Neo4j + ChromaDB) and Ollama running while you chat.

<br>

## Data Flow

### Ingest Path

```
text + scope
     │
     ▼
┌──────────┐     ┌──────────────────────┐
│  Ollama  │────▶│  Extract Triples     │──── (subject, predicate, object)
│          │     │  Generate Embedding   │──── float64 vector
└──────────┘     └──────────────────────┘
                          │
               ┌──────────┴──────────┐
               ▼                     ▼
        ┌─────────────┐      ┌─────────────┐
        │  ChromaDB   │      │   Neo4j     │
        │  store doc  │      │  MERGE rels │
        └─────────────┘      └─────────────┘
            (parallel — two goroutines, error channel)
```

### Recall Path

```
query + intent + scope filter
     │
     ▼
  Embed query
     │
     ├───── goroutine ──▶  ChromaDB vector search (oversampled 2×)
     │
     ├───── goroutine ──▶  Neo4j graph traversal (intent or related)
     │
     ▼  (sync.WaitGroup)
  Merge results ──▶ Graph boost (1.5×) on dual hits
     │
     ▼
  Temporal decay ──▶ score × e^(−λ × hours)
     │
     ▼
  Frequency suppression ──▶ skip if over-injected this session
     │
     ▼
  Negative weights ──▶ multiply by suppression weight, drop if ≤ 0.001
     │
     ▼
  Sort · Trim to top_k · Record injection counts
```

<br>

## MCP Tool Reference

### `ingest_memory`
Store a memory. Triples are extracted automatically; embedding is generated and stored.

```json
{
  "text": "I'm building a CLI tool in Go called Architect",
  "scope": "/projects/architect-cli",
  "metadata": {}
}
```
Returns: `memory_id`, `scope`, `triples_extracted`

---

### `recall_memory`
Hybrid retrieval with all ranking stages applied.

```json
{
  "query": "CLI tools I've been working on",
  "intent": "find active projects",
  "top_k": 5,
  "cross_scope": false
}
```
Returns: ranked memories with scores, or `inhibited: true` if session is in clean-slate mode.

---

### `new_session`
Start a new session. Optionally inhibit all past context or pre-mount scopes.

```json
{ "inhibit": true, "mount_scopes": ["/projects/architect-cli"] }
```

---

### `suppress_topic`
Temporarily suppress a topic from recall. Decays each turn.

```json
{ "term": "C++", "ttl": 50, "weight": 0.0 }
```

---

### `forget_memory`
Permanent deletion from both vector and graph stores.

```json
{ "memory_id": "uuid-here" }
```

---

### `mount_scope` / `unmount_scope`
Control scope visibility for the current session.

```json
{ "scope_path": "/projects/architect-cli" }
```

---

### `list_scopes`
Returns all scopes with mount status and memory counts.

### `memory_status`
Full diagnostic — session state, mounted scopes, active suppressions, store statistics.

<br>

## MCP Resources

| URI | Description |
|:----|:------------|
| `memory://profile` | Plain-text user memory profile — session ID, mounted scopes, and graph-derived relationship triples. |

<br>

## Configuration

All configuration is via environment variables with the `GRAPHITE_` prefix.

<details>
<summary><strong>Full variable reference</strong></summary>

<br>

| Variable | Default | Description |
|:---|:---|:---|
| `GRAPHITE_TRANSPORT` | `stdio` | Transport mode: `stdio` or `sse` |
| `GRAPHITE_SSE_ADDR` | `:3100` | HTTP listen address (SSE mode) |
| `GRAPHITE_CHROMA_URL` | `http://localhost:8000` | ChromaDB endpoint |
| `GRAPHITE_NEO4J_URI` | `bolt://localhost:7687` | Neo4j Bolt URI |
| `GRAPHITE_NEO4J_USER` | `neo4j` | Neo4j username |
| `GRAPHITE_NEO4J_PASS` | `graphite` | Neo4j password |
| `GRAPHITE_OLLAMA_URL` | `http://localhost:11434` | Ollama endpoint |
| `GRAPHITE_OLLAMA_MODEL` | `llama3.1` | Model for triple extraction + embedding |
| `GRAPHITE_DECAY_LAMBDA` | `0.01` | Decay constant λ (per hour): `score × e^(−λ·hours)`. Set to `0` to disable decay. Ignored if `GRAPHITE_DECAY_HALF_LIFE_DAYS` is set. |
| `GRAPHITE_DECAY_HALF_LIFE_DAYS` | *(unset)* | If set to a positive number, λ is computed as `ln(2) / (days × 24)` and overrides `GRAPHITE_DECAY_LAMBDA`. |
| `GRAPHITE_SUPPRESS_THRESHOLD` | `3` | Injection count before frequency cooldown |
| `GRAPHITE_SUPPRESS_COOLDOWN` | `5` | Turns to hide a frequently injected fact |
| `GRAPHITE_DEFAULT_SCOPE` | `/default` | Default memory scope path |
| `GRAPHITE_NEG_WEIGHT_DEFAULT_TTL` | `50` | Default suppression TTL in turns |

</details>

<br>

## Project Structure

```
graphite-mem/
├── cmd/graphite-mem/
│   └── main.go              # Wiring, transport selection, server bootstrap
├── internal/
│   ├── config/              # Env-based configuration loader
│   ├── llm/                 # Ollama HTTP client (triples + embeddings)
│   ├── ingestor/            # Parallel dual-store ingest pipeline
│   ├── governor/            # Hybrid retrieval engine + all ranking stages
│   ├── vault/               # Sessions, scopes, negative weights, inhibit
│   └── storage/             # ChromaDB + Neo4j drivers with scope filtering
├── tools/                   # MCP tool definitions (9 tools)
├── resources/               # MCP resource: memory://profile, readme-banner.svg
├── scripts/
│   ├── docker-compose.yml   # Neo4j 5 + ChromaDB containers
│   └── seed.cypher          # Optional seed data
├── Makefile                 # build · run · test · lint · docker-up/down/reset
└── README.md
```

<br>

## Design Decisions

<table>
<tr>
<td width="30%"><strong>Why Go?</strong></td>
<td>Goroutines make parallel vector + graph queries trivial. Recall merges both result sets in sub-15ms. No GC pauses keeps time-to-first-token low for the LLM client.</td>
</tr>
<tr>
<td><strong>Why Graph + Vector?</strong></td>
<td>Vector search is a proximity lookup — it finds <em>what</em> is similar. Graph traversal finds <em>why</em> it matters: intent chains, project relationships, goal hierarchies. Combining them produces recall that feels like understanding.</td>
</tr>
<tr>
<td><strong>Why Scoped Memory?</strong></td>
<td>Flat memory pollutes. Asking about your Go CLI shouldn't surface your Python ML notes. Scopes are path-based (<code>/projects/foo</code>) and mount/unmount instantly — no re-indexing, no copying.</td>
</tr>
<tr>
<td><strong>Why Frequency Suppression?</strong></td>
<td>Without it, the same high-similarity memory gets injected every turn. The suppressor tracks injection counts per session and applies a cooldown window — the memory comes back after K turns, not gone forever.</td>
</tr>
<tr>
<td><strong>Why Negative Weights?</strong></td>
<td>Sometimes you need the AI to <em>not</em> bring something up. Negative weights apply a score multiplier (down to 0) on substring-matched terms, with TTL-based decay so they expire naturally.</td>
</tr>
</table>

<br>

## Makefile

```bash
make build          # → bin/graphite-mem
make run            # stdio mode
make run-sse        # HTTP/SSE mode on :3100
make test           # go test ./... -v -race
make lint           # go vet ./...
make docker-up      # start Neo4j + ChromaDB
make docker-down    # stop containers
make docker-reset   # nuke volumes and restart
make tidy           # go mod tidy
make clean          # rm -rf bin/
```

<br>

<div align="center">

---

<sub>Built with Go · Chroma · Neo4j · Ollama · MCP</sub>

MIT License

</div>
