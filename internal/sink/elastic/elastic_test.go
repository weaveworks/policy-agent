package elastic

import (
	"context"
	"testing"
	"time"

	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/stretchr/testify/assert"
)

const (
	address   = "http://localhost:9200"
	indexName = "test_audit_validation"
)

func TestWriteElasticsearchSink(t *testing.T) {
	schemaFilePath = "schema.json"

	var auditEvents []domain.PolicyValidation
	documentsCount := 4

	for i := 0; i < documentsCount; i++ {
		auditEvents = append(auditEvents, GeneratePolicyValidationObject())
	}

	sink, err := NewElasticSearchSink(
		address, "", "", indexName,
	)
	if err != nil {
		t.Error("Error initializing elasticsearch sink")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Start(ctx)
	sink.Write(ctx, auditEvents)
	time.Sleep(12 * time.Second)

	got, err := getCount(sink.elasticClient, sink.indexName)
	if err != nil {
		t.Error("Error getting index count")
	}
	assert.Equal(t, documentsCount, got, "Error getting index count")
}
