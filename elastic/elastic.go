package elastic

import (
	"context"

	"github.com/JIeeiroSst/utils/logger"
	"github.com/olivere/elastic/v7"
)

func NewElastic(dns string) *elastic.Client {
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(dns))
	if err != nil {
		logger.Error(context.Background(), "NewElastic err %v", err)
	}

	return client
}
