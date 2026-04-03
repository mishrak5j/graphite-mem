package ingestor

import (
	"github.com/mishrak5j/graphite-mem/internal/llm"
	"github.com/mishrak5j/graphite-mem/internal/storage"
)

func llmTriplesToStorage(triples []llm.Triple) []storage.Triple {
	result := make([]storage.Triple, len(triples))
	for i, t := range triples {
		result[i] = storage.Triple{
			Subject:   t.Subject,
			Predicate: t.Predicate,
			Object:    t.Object,
		}
	}
	return result
}
