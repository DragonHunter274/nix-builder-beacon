package builder

import (
	"slices"
	"sync"
)

// Index is a goroutine-safe set of discovered builders, keyed by mDNS instance name.
type Index struct {
	mux      sync.RWMutex
	builders map[string]*Builder
}

func NewIndex() *Index {
	return &Index{
		builders: make(map[string]*Builder),
	}
}

// Add inserts or replaces the builder with the given ID.
func (idx *Index) Add(b *Builder) {
	idx.mux.Lock()
	defer idx.mux.Unlock()

	idx.builders[b.ID] = b
}

// Remove deletes the builder with the given ID.
func (idx *Index) Remove(id string) {
	idx.mux.Lock()
	defer idx.mux.Unlock()

	delete(idx.builders, id)
}

// Snapshot returns all known builders, sorted by hostname for deterministic output.
func (idx *Index) Snapshot() []*Builder {
	idx.mux.RLock()
	defer idx.mux.RUnlock()

	out := make([]*Builder, 0, len(idx.builders))
	for _, b := range idx.builders {
		out = append(out, b)
	}

	slices.SortFunc(out, func(a, b *Builder) int {
		if a.Hostname != b.Hostname {
			if a.Hostname < b.Hostname {
				return -1
			}
			return 1
		}
		return 0
	})

	return out
}
