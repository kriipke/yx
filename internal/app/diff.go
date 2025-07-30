package app

import (
    "gopkg.in/yaml.v3"
    "github.com/kriipke/yiff/pkg/differ"     // Use diff logic from pkg/differ
    "github.com/kriipke/yiff/internal/core"  // Use your app's types, if needed
)

// Option 2: Wrap the output in a core.DiffResult struct
func DiffYAMLResult(a, b []byte) (*core.DiffResult, error) {
    diffs, err := DiffYAML(a, b)
    if err != nil {
        return nil, err
    }
    return &core.DiffResult{Variables: diffs}, nil
}
