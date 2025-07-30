package core

type VariableDiff struct {
	Name    string      `json:"name" yaml:"name"`
	Default interface{} `json:"default" yaml:"default"`
	Value   interface{} `json:"value" yaml:"value"`
	Status  string      `json:"status" yaml:"status"` // "changed", "added", "removed"
}
