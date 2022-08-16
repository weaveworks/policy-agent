# ElasticSearch Sink
The policy agent supports multiple methods to expose this data, one of them is ElasticSearch Sink.
The results of validating entities against policies would be written in ElasticSearch index with validation objects [schema](schema.json). It could be configured differently in different agent modes: `Audit` or `Admission`

## ElasticSearch configuration:

- address: ElasticSearch server address

- indexName: Index name the results would be written in

- username: User credentials to access ElasticSearch service

- password: User credentials to access ElasticSearch service

- insertionMode: It could be a choice of both `insert` or `upsert`, it defines the way the document is written.
			 
	#### Insertion modes
	1. Insert mode: would give an insight of all the historical data, doesn't update or delete any old records. so the index would contain a log for all validation objects.
	2. Upsert mode: Would update the old result of validating an entity against a policy happens in the same day, so the index would only contain the latest validation results for a policy and entity combination per day.

To enable writing validation objects in ElasticSearch for Audit mode:
```yaml
config:
	audit:
		sinks:
			elasticSink:
				address: ""
				username: ""
				password: ""
				indexName: ""
				insertionMode: "upsert"
```
To enable writing validation objects in elasticsearch for Admission mode:
```yaml
config:
	admission:
		sinks:
			elasticSink:
				address: ""
				username: ""
				password: ""
				indexName: ""
				insertionMode: "upsert"
```