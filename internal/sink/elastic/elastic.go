package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

const (
	resultChanSize  int           = 50
	batchSize       int           = 100
	batchExpiry     time.Duration = 10 * time.Second
	retriesInterval time.Duration = 500 * time.Millisecond
	retries         int           = 5
)

var schemaFilePath string = "internal/sink/elastic/schema.json"

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
	InsertionMode          string
}

// NewElasticSearchSink returns a sink that sends results to elasticsearch index
func NewElasticSearchSink(address, username, password, index, insertionMode string) (*ElasticSearchSink, error) {
	client, err := elasticsearch.NewClient(
		elasticsearch.Config{
			Addresses: []string{address},
			Username:  username,
			Password:  password,
		},
	)
	if err != nil {
		return nil, err
	}

	err = createIndexSchema(client, index)
	if err != nil {
		return nil, err
	}

	return &ElasticSearchSink{
		policyValidationChan:   make(chan domain.PolicyValidation, resultChanSize),
		policyValidationsBatch: make([]domain.PolicyValidation, 0, batchSize),
		elasticClient:          client,
		indexName:              index,
		InsertionMode:          insertionMode,
	}, nil
}

// Write adds results to buffer, implements github.com/MagalixTechnologies/policy-core/domain.PolicyValidationSink
func (es *ElasticSearchSink) Write(_ context.Context, policyValidations []domain.PolicyValidation) error {
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
		body, err := createIndexBody(items, es.indexName, es.InsertionMode)
		if err != nil {
			logger.Errorw("failed to create policy validation elastic search body", "error", err)
			continue
		}
		res, err := es.elasticClient.Bulk(bytes.NewReader(body))
		if err != nil || res.StatusCode != 200 {
			logger.Warnw("failed to write policy validations", "index", es.indexName, "retry", i+1, "error", err)
			continue
		}
		defer res.Body.Close()
		return
	}
	time.Sleep(retriesInterval)
	logger.Errorw("failed to write policy validations", "index", es.indexName, "error", err)
}

func createIndexSchema(client *elasticsearch.Client, index string) error {
	response, err := client.Indices.Exists([]string{index})
	if err != nil {
		return errors.WithMessagef(err, "failed to check if index exists")
	}
	if response.StatusCode == http.StatusNotFound {
		response, err = client.Indices.Create(index)
		if err != nil || response.StatusCode != http.StatusOK {
			return errors.WithMessagef(err, "failed to create index")
		}
		logger.Infof("index %s is created", index)
	}
	//internal/sink/elastic/
	schema, err := ioutil.ReadFile(schemaFilePath)
	if err != nil {
		return err
	}

	response, err = client.Indices.PutMapping(bytes.NewReader(schema), client.Indices.PutMapping.WithIndex(index))
	if err != nil || response.StatusCode != http.StatusOK {
		return errors.WithMessagef(err, "failed to update schema")
	}
	return nil
}

func createIndexBody(items []domain.PolicyValidation, index, insertionMode string) ([]byte, error) {
	var body []byte
	for _, item := range items {
		id := item.ID
		if insertionMode == "upsert" {
			id = fmt.Sprintf("%s_%s_%s", item.Policy.ID, item.Entity.ID, item.CreatedAt.Format("2006-02-01"))
		}
		itemBody, err := createDocumentBody(item, id, index)
		if err != nil {
			return nil, err
		}
		body = append(body, itemBody...)
	}
	return body, nil
}

func createDocumentBody(document interface{}, id string, index string) ([]byte, error) {
	header, err := json.Marshal(IndexTemplate{Index: Index{IndexName: index, Id: id}})
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
