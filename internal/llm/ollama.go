package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type generateResponse struct {
	Response string `json:"response"`
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

const triplePrompt = `Extract (Subject, Predicate, Object) triples from the following text.
Return ONLY a JSON array of objects with keys "s", "p", "o".
Example: [{"s":"Alice","p":"works_at","o":"Google"}]
If no clear triples can be extracted, return an empty array [].

Text: %s`

func (c *OllamaClient) ExtractTriples(ctx context.Context, text string) ([]Triple, error) {
	prompt := fmt.Sprintf(triplePrompt, text)

	body, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama generate: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama generate status %d: %s", resp.StatusCode, string(respBody))
	}

	var genResp generateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return nil, fmt.Errorf("parse generate response: %w", err)
	}

	return parseTriples(genResp.Response)
}

func parseTriples(raw string) ([]Triple, error) {
	raw = strings.TrimSpace(raw)
	// Strip markdown code fences if present
	if strings.HasPrefix(raw, "```") {
		if i := strings.Index(raw, "\n"); i >= 0 {
			raw = strings.TrimSpace(raw[i+1:])
		}
		if i := strings.LastIndex(raw, "```"); i > 0 {
			raw = strings.TrimSpace(raw[:i])
		}
	}

	// Prefer structure by first byte — do not scan for '[' (can appear inside string values).
	switch {
	case strings.HasPrefix(raw, "{"):
		var one Triple
		if err := json.Unmarshal([]byte(raw), &one); err == nil {
			if one.Subject == "" && one.Predicate == "" && one.Object == "" {
				return nil, nil
			}
			return []Triple{one}, nil
		}
	case strings.HasPrefix(raw, "["):
		var triples []Triple
		if err := json.Unmarshal([]byte(raw), &triples); err == nil {
			return triples, nil
		}
	default:
		start := strings.Index(raw, "[")
		end := strings.LastIndex(raw, "]")
		if start >= 0 && end > start {
			trim := raw[start : end+1]
			var triples []Triple
			if err := json.Unmarshal([]byte(trim), &triples); err == nil {
				return triples, nil
			}
		}
	}

	var triples []Triple
	if err := json.Unmarshal([]byte(raw), &triples); err != nil {
		var one Triple
		if err2 := json.Unmarshal([]byte(raw), &one); err2 == nil && (one.Subject != "" || one.Predicate != "" || one.Object != "") {
			return []Triple{one}, nil
		}
		var wrapper map[string]json.RawMessage
		if err2 := json.Unmarshal([]byte(raw), &wrapper); err2 == nil {
			for _, v := range wrapper {
				if err3 := json.Unmarshal(v, &triples); err3 == nil {
					return triples, nil
				}
			}
		}
		return nil, fmt.Errorf("parse triples from: %s: %w", raw, err)
	}
	return triples, nil
}

func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{
		Model: c.model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed status %d: %s", resp.StatusCode, string(respBody))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("parse embed response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	f64 := embedResp.Embeddings[0]
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32, nil
}
