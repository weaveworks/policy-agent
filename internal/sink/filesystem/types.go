package filesystem

import "time"

type Result struct {
	ID              string                 `json:"id"`
	AccountID       string                 `json:"account_id"`
	ClusterID       string                 `json:"cluster_id"`
	PolicyID        string                 `json:"policy_id"`
	Status          string                 `json:"status"`
	Type            string                 `json:"type"`
	Provider        string                 `json:"provider"`
	EntityName      string                 `json:"entity_name"`
	EntityType      string                 `json:"entity_type"`
	EntityNamespace string                 `json:"entity_namespace"`
	CreatedAt       time.Time              `json:"created_at"`
	Message         string                 `json:"message"`
	Info            map[string]interface{} `json:"info"`
	CategoryID      string                 `json:"category_id"`
	Severity        string                 `json:"severity"`
}
