package elastic

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

type ElasticsearchClient struct {
	Client   *elasticsearch.Client
	Document ElasticsearchDocument
}

type ElasticsearchDocument struct {
	ID       string `json:"id"`
	Index    string
	Mapping  string
	Document []byte
}

type ElasticsearchAccount struct {
	Addresses []string `json:"addresses"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
}

func NewElasticsearchClient(account ElasticsearchAccount, document ElasticsearchDocument) (*ElasticsearchClient, error) {
	cfg := elasticsearch.Config{
		Addresses: account.Addresses,
		Username:  account.Username,
		Password:  account.Password,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Elasticsearch client: %v", err)
	}

	reqIndex := esapi.IndicesCreateRequest{
		Index: document.Index,
		Body:  strings.NewReader(document.Mapping),
	}

	res, err := reqIndex.Do(context.Background(), es)
	if err != nil {
		return nil, fmt.Errorf("error creating index: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.Status())
	}

	reqDocument := esapi.IndexRequest{
		Index:      document.Index,
		DocumentID: document.ID,
		Body:       strings.NewReader(string(document.Document)),
	}

	res, err = reqDocument.Do(context.Background(), es)
	if err != nil {
		return nil, fmt.Errorf("error indexing document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.Status())
	}

	return &ElasticsearchClient{
		Client:   es,
		Document: document,
	}, nil
}
