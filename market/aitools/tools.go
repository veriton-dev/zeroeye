// Package ai — tool hooks for market AI assistants.
// Scaffolding for read-only market data tools agents can call.
package aitools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ToolKind classifies what a tool is allowed to do.
type ToolKind string

const (
	ToolKindRead  ToolKind = "read"
	ToolKindWrite ToolKind = "write" // reserved; not used by read hooks
)

// ToolSpec describes a single AI-callable tool.
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Kind        ToolKind        `json:"kind"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// ToolResult is a normalized tool invocation response.
type ToolResult struct {
	OK        bool           `json:"ok"`
	Tool      string         `json:"tool"`
	Data      map[string]any `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
	TookMS    int64          `json:"took_ms"`
	InvokedAt time.Time      `json:"invoked_at"`
}

// ToolHandler executes a tool by name with JSON args.
type ToolHandler func(args map[string]any) (map[string]any, error)

// ToolRegistry holds read (and future write) tool hooks.
type ToolRegistry struct {
	mu       sync.RWMutex
	specs    map[string]ToolSpec
	handlers map[string]ToolHandler
}

// NewToolRegistry returns an empty registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		specs:    make(map[string]ToolSpec),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds or replaces a tool.
func (r *ToolRegistry) Register(spec ToolSpec, h ToolHandler) error {
	if r == nil {
		return fmt.Errorf("nil registry")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return fmt.Errorf("tool name required")
	}
	if h == nil {
		return fmt.Errorf("handler required for %s", name)
	}
	if spec.Kind == "" {
		spec.Kind = ToolKindRead
	}
	spec.Name = name
	r.mu.Lock()
	defer r.mu.Unlock()
	r.specs[name] = spec
	r.handlers[name] = h
	return nil
}

// List returns registered tool specs sorted by name.
func (r *ToolRegistry) List() []ToolSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolSpec, 0, len(r.specs))
	for _, s := range r.specs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Invoke runs a tool by name.
func (r *ToolRegistry) Invoke(name string, args map[string]any) ToolResult {
	start := time.Now()
	res := ToolResult{Tool: name, InvokedAt: start}
	if r == nil {
		res.Error = "nil registry"
		res.TookMS = time.Since(start).Milliseconds()
		return res
	}
	r.mu.RLock()
	h, ok := r.handlers[name]
	r.mu.RUnlock()
	if !ok {
		res.Error = fmt.Sprintf("unknown tool %q", name)
		res.TookMS = time.Since(start).Milliseconds()
		return res
	}
	if args == nil {
		args = map[string]any{}
	}
	data, err := h(args)
	res.TookMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.OK = true
	res.Data = data
	return res
}

// DefaultReadMarketParams is the JSON Schema for read_market_data.
const DefaultReadMarketParams = `{
  "type": "object",
  "properties": {
    "symbol": {"type": "string", "description": "Market symbol, e.g. BTC-USD"},
    "fields": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Optional subset: bid, ask, mid, volume, ts"
    }
  },
  "required": ["symbol"],
  "additionalProperties": false
}`

// MarketQuoteSource supplies read-only quotes for the tool hook.
// Implementations may wrap orderbook, pricing, or a stub for tests.
type MarketQuoteSource interface {
	Quote(symbol string) (map[string]any, error)
}

// StubQuoteSource returns deterministic demo quotes (scaffolding).
type StubQuoteSource struct{}

// Quote implements MarketQuoteSource.
func (StubQuoteSource) Quote(symbol string) (map[string]any, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return nil, fmt.Errorf("symbol required")
	}
	// Deterministic pseudo-quote so agents get stable scaffolding output.
	base := float64(len(sym))*10.0 + 100.0
	return map[string]any{
		"symbol": sym,
		"bid":    base - 0.05,
		"ask":    base + 0.05,
		"mid":    base,
		"volume": float64(1000 + len(sym)*17),
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"source": "stub_quote_v1",
	}, nil
}

// RegisterReadMarketData installs the read_market_data tool hook.
func (r *ToolRegistry) RegisterReadMarketData(src MarketQuoteSource) error {
	if src == nil {
		src = StubQuoteSource{}
	}
	spec := ToolSpec{
		Name:        "read_market_data",
		Description: "Read-only market quote snapshot for a symbol (bid/ask/mid/volume). No orders, no side effects.",
		Kind:        ToolKindRead,
		Parameters:  json.RawMessage(DefaultReadMarketParams),
	}
	return r.Register(spec, func(args map[string]any) (map[string]any, error) {
		raw, _ := args["symbol"].(string)
		q, err := src.Quote(raw)
		if err != nil {
			return nil, err
		}
		// Optional field filter
		if fields, ok := args["fields"].([]any); ok && len(fields) > 0 {
			filtered := map[string]any{"symbol": q["symbol"]}
			for _, f := range fields {
				name, _ := f.(string)
				if name == "" || name == "symbol" {
					continue
				}
				if v, exists := q[name]; exists {
					filtered[name] = v
				}
			}
			return filtered, nil
		}
		return q, nil
	})
}

// DefaultTools returns a registry with read_market_data pre-registered.
func DefaultTools() *ToolRegistry {
	r := NewToolRegistry()
	_ = r.RegisterReadMarketData(StubQuoteSource{})
	return r
}
