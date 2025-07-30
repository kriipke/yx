package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kriipke/yiff/pkg/differ"
)

func DiffHandler(w http.ResponseWriter, r *http.Request) {
	// Expect JSON: {"a": "<yaml string>", "b": "<yaml string>"}
	var input struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	yamlA, err := differ.LoadYAMLMap([]byte(input.A))
	if err != nil {
		http.Error(w, "Invalid YAML for 'a': "+err.Error(), http.StatusBadRequest)
		return
	}
	yamlB, err := differ.LoadYAMLMap([]byte(input.B))
	if err != nil {
		http.Error(w, "Invalid YAML for 'b': "+err.Error(), http.StatusBadRequest)
		return
	}
	diffs := differ.Diff(yamlA, yamlB)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"variables": diffs,
	})
}
