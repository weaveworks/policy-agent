package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/weaveworks/policy-agent/pkg/logger"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	jsonPathArrRegex = regexp.MustCompile("^([a-zA-Z0-9]+)\\[([0-9]+)\\]")
)

const (
	mutatedLabel = "pac.weave.works/mutated"
)

type MutationResult struct {
	raw  []byte
	node *yaml.RNode
}

// NewMutationResult create new MutationResult object
func NewMutationResult(entity Entity) (*MutationResult, error) {
	raw, err := json.Marshal(entity.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity %s. error: %w", entity.Name, err)
	}

	var ynode yaml.Node
	err = yaml.Unmarshal(raw, &ynode)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal entity %s. error: %w", entity.Name, err)
	}

	return &MutationResult{
		raw:  raw,
		node: yaml.NewRNode(&ynode),
	}, nil
}

// Mutate mutate resource by applying the recommended values of the given occurrences
func (m *MutationResult) Mutate(occurrences []Occurrence) ([]Occurrence, error) {
	var mutated bool
	for i, occurrence := range occurrences {
		if occurrence.ViolatingKey == nil || occurrence.RecommendedValue == nil {
			continue
		}

		path := parseKeyPath(*occurrence.ViolatingKey)
		pathGetter := yaml.LookupCreate(yaml.MappingNode, path...)
		node, err := m.node.Pipe(pathGetter)
		if err != nil {
			logger.Errorw("failed while getting field's node", "error", err)
			continue
		}

		if node == nil {
			logger.Errorw("field not found", "path", occurrence.ViolatingKey)
			continue
		}

		value := occurrence.RecommendedValue
		if number, ok := value.(json.Number); ok {
			value, err = number.Float64()
			if err != nil {
				logger.Errorw("failed to parse number", "error", err)
				continue
			}
		}

		err = node.Document().Encode(value)
		if err != nil {
			logger.Errorw("failed to encode recommended value", "path", occurrence.ViolatingKey, "value", occurrence.RecommendedValue)
			continue
		}

		occurrences[i].Mutated = true
		mutated = true
	}
	if mutated {
		labels := m.node.GetLabels()
		labels[mutatedLabel] = ""
		m.node.SetLabels(labels)
	}
	return occurrences, nil
}

// OldResource return old resource before mutation
func (m *MutationResult) OldResource() []byte {
	return m.raw
}

// NewResource return mutated resource
func (m *MutationResult) NewResource() ([]byte, error) {
	return m.node.MarshalJSON()
}

func parseKeyPath(path string) []string {
	var keys []string
	parts := strings.Split(path, ".")
	for _, part := range parts {
		groups := jsonPathArrRegex.FindStringSubmatch(part)
		if groups == nil {
			keys = append(keys, part)
		} else {
			keys = append(keys, groups[1:]...)
		}
	}
	return keys
}
