package elastic

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/uuid-go"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

func getCount(client *elasticsearch.Client, index string) (int, error) {
	res, err := client.Count(client.Count.WithIndex(index))
	if err != nil || res.StatusCode != 200 {
		return 0, errors.WithMessagef(err, "Cannot get index count")
	}

	var response struct {
		Count int `json:"count"`
	}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return 0, errors.WithMessagef(err, "failed to decode index count response")
	}
	return response.Count, nil
}

// RandomText generates random name
func RandomText() string {
	return strings.ReplaceAll(uuid.NewV4().String(), "-", "")
}

// RandomUUID generates random uuid
func RandomUUID() string {
	return uuid.NewV4().String()
}

func GeneratePolicyValidationObject() domain.PolicyValidation {
	violatingkey := RandomText()
	return domain.PolicyValidation{
		ID:        RandomUUID(),
		AccountID: RandomUUID(),
		ClusterID: RandomUUID(),
		Policy: domain.Policy{
			Name:        RandomText(),
			ID:          RandomUUID(),
			Description: RandomText(),
			HowToSolve:  RandomText(),
			Category:    RandomText(),
			Tags:        []string{RandomText()},
			Severity:    RandomText(),
			Standards: []domain.PolicyStandard{
				{
					ID:       RandomText(),
					Controls: []string{RandomText()},
				},
			},
		},
		Entity: domain.Entity{
			ID:         RandomUUID(),
			Name:       RandomText(),
			Kind:       RandomText(),
			Namespace:  RandomText(),
			APIVersion: RandomText(),
			Manifest: map[string]interface{}{
				"test": RandomText(),
			},
		},
		Occurrences: []domain.Occurrence{
			{
				Message:          RandomText(),
				ViolatingKey:     &violatingkey,
				RecommendedValue: false,
			},
		},
		Status:    RandomText(),
		Trigger:   RandomText(),
		CreatedAt: time.Now().Round(0),
	}
}
