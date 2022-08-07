package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/elastic/go-elasticsearch/v7"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	resultChanSize  int           = 50
	batchExpiry     time.Duration = 10 * time.Second
	retriesInterval time.Duration = 500 * time.Millisecond
	batchSize       int           = 100
	retries         int           = 5
)

type IndexTemplate struct {
	Index Index `json:"index"`
}

type Index struct {
	IndexName string `json:"_index"`
	Id        string `json:"_id"`
}

type ElasticSearchSink struct {
	policyValidationChan   chan domain.PolicyValidation
	elasticClient          *elasticsearch.Client
	indexName              string
	policyValidationsBatch []domain.PolicyValidation
}

// NewElasticSearchSink returns a sink that sends results to elasticsearch index
func NewElasticSearchSink(address, username, password, index string) (*ElasticSearchSink, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{address},
		Username:  username,
		Password:  password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ElasticSearchSink{
		policyValidationChan:   make(chan domain.PolicyValidation, resultChanSize),
		policyValidationsBatch: make([]domain.PolicyValidation, 0, batchSize),
		elasticClient:          client,
		indexName:              index,
	}, nil
}

// Write adds results to buffer, implements github.com/MagalixTechnologies/policy-core/domain.PolicyValidationSink
func (es *ElasticSearchSink) Write(_ context.Context, policyValidations []domain.PolicyValidation) error {
	// logger.Infow("writing validation results", "sink", "elasticsearch", "count", len(policyValidations))
	for _, policyValidation := range policyValidations {
		es.policyValidationChan <- policyValidation
	}
	return nil
}

// Start starts the sink to send events when batch size is met or an interval has passed
func (es *ElasticSearchSink) Start(ctx context.Context) error {
	timer := time.NewTicker(batchExpiry)

	for {
		select {
		case result := <-es.policyValidationChan:
			es.policyValidationsBatch = append(es.policyValidationsBatch, result)
			if len(es.policyValidationsBatch) == cap(es.policyValidationsBatch) {
				es.writeBatch(es.policyValidationsBatch)
				es.policyValidationsBatch = es.policyValidationsBatch[:0]
				timer.Reset(batchExpiry)
			}
		case <-timer.C:
			if len(es.policyValidationsBatch) > 0 {
				es.writeBatch(es.policyValidationsBatch)
				es.policyValidationsBatch = es.policyValidationsBatch[:0]
			}
		case <-ctx.Done():
			if len(es.policyValidationsBatch) > 0 {
				es.writeBatch(es.policyValidationsBatch)
			}
			return ctx.Err()
		}
	}
}

func (es *ElasticSearchSink) writeBatch(items []domain.PolicyValidation) {
	var err error
	logger.Infow("writing policy validations", "size", len(items), "index", es.indexName)

	for i := 0; i < retries; i++ {
		var body []byte
		for _, item := range items {
			itemBody, err := createIndexBody(item, es.indexName)
			if err != nil {
				logger.Errorw("failed to create policy validation elastic search body", item, "error", err)
			}
			body = append(body, itemBody...)
		}
		res, err := es.elasticClient.Bulk(bytes.NewReader(body))
		if err != nil || res.StatusCode != 200 {
			logger.Warnw("failed to write policy validations", "index", es.indexName, "retry", i+1, "error", err)
		} else {
			return
		}
		defer res.Body.Close()
	}
	time.Sleep(retriesInterval)
	logger.Errorw("failed to write policy validations", "index", es.indexName, "error", err)
}

func createIndexBody(document interface{}, index string) ([]byte, error) {
	header, err := json.Marshal(IndexTemplate{Index: Index{IndexName: index, Id: string(uuid.NewUUID())}})
	if err != nil {
		return nil, err
	}
	indexBody, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	reqBody := append(header, []byte("\n")...)
	reqBody = append(reqBody, indexBody...)
	reqBody = append(reqBody, []byte("\n")...)

	return reqBody, nil
}
