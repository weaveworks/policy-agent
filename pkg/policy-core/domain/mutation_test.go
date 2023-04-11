package domain

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMutation(t *testing.T) {
	violationKey1 := "spec.template.spec.containers[0].securityContext.privileged"
	violationKey2 := "metadata.labels.owner"

	tests := []struct {
		entityFile           string
		mutatedEntityFile    string
		Occurrences          []Occurrence
		FixedOccurrenceCount int
	}{
		{
			entityFile:        "testData/entity-1.yaml",
			mutatedEntityFile: "testData/mutated-entity-1.yaml",
			Occurrences: []Occurrence{
				{
					ViolatingKey:     &violationKey1,
					RecommendedValue: false,
				},
				{
					ViolatingKey:     &violationKey2,
					RecommendedValue: "test",
				},
			},
			FixedOccurrenceCount: 2,
		},
	}

	for _, tt := range tests {
		entity, err := getEntityFromFile(tt.entityFile)
		assert.Nil(t, err)

		result, err := NewMutationResult(entity)
		assert.Nil(t, err)

		occurrences, err := result.Mutate(tt.Occurrences)
		assert.Nil(t, err)

		var fixedOccurrenceCount int
		for i := range occurrences {
			if occurrences[i].Mutated {
				fixedOccurrenceCount++
			}
		}

		mutated, err := result.NewResource()
		assert.Nil(t, err)

		expectedMutatedEntity, err := ioutil.ReadFile(tt.mutatedEntityFile)
		assert.Nil(t, err)

		assert.YAMLEq(t, string(expectedMutatedEntity), string(mutated))
	}

}

func getEntityFromFile(path string) (Entity, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return Entity{}, err
	}

	var m map[string]interface{}
	err = yaml.Unmarshal(raw, &m)
	if err != nil {
		return Entity{}, err
	}

	return NewEntityFromSpec(m), nil
}
