package elastic

import (
	"github.com/olivere/elastic/v7"
)

func NewElastic(dns string) *elastic.Client {
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(dns))
	if err != nil {
		panic(err)
	}

	return client
}
