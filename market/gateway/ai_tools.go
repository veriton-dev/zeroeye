package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/tent-of-trials/market/aitools"
)

var (
	aiToolsOnce sync.Once
	aiTools     *aitools.ToolRegistry
)

func getAITools() *aitools.ToolRegistry {
	aiToolsOnce.Do(func() {
		aiTools = aitools.DefaultTools()
	})
	return aiTools
}

// HandleAIToolsList GET /api/v1/ai/tools
func HandleAIToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"tools": getAITools().List(),
	})
}

// HandleAIToolInvoke POST /api/v1/ai/tools/{name}
func HandleAIToolInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/ai/tools/")
	name = strings.Trim(name, "/")
	if name == "" || strings.Contains(name, "/") {
		http.Error(w, "tool name required", http.StatusBadRequest)
		return
	}
	var args map[string]any
	dec := json.NewDecoder(r.Body)
	_ = dec.Decode(&args)
	res := getAITools().Invoke(name, args)
	status := http.StatusOK
	if !res.OK {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}

// mountAIToolRoutes registers AI read-tool hook endpoints.
func (g *Gateway) mountAIToolRoutes() {
	if g == nil || g.mux == nil {
		return
	}
	g.mux.HandleFunc("/api/v1/ai/tools", HandleAIToolsList)
	g.mux.HandleFunc("/api/v1/ai/tools/", HandleAIToolInvoke)
}
