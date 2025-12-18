package cloudprovider

import (
	"fmt"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/model"
)

type ResourceCleaner interface {
	Do() error
}

//go:generate mockery --name=ProviderFactory
type ProviderFactory interface {
	New(hyperscalerType model.HyperscalerType, secretData map[string][]byte, market model.Market) (ResourceCleaner, error)
}

type providerFactory struct{}

func NewProviderFactory() ProviderFactory {
	return &providerFactory{}
}

func (pf *providerFactory) New(hyperscalerType model.HyperscalerType, secretData map[string][]byte, market model.Market) (ResourceCleaner, error) {
	switch hyperscalerType {
	case model.GCP:
		{
			return NewGCPeResourcesCleaner(secretData), nil
		}
	case model.Azure:
		{
			return NewAzureResourcesCleaner(secretData, market)
		}
	case model.AWS:
		{
			return NewAwsResourcesCleaner(secretData)
		}
	case model.Alicloud:
		{
			return NewAliCloudResourcesCleaner(secretData), nil
		}
	default:
		return nil, fmt.Errorf("unknown hyperscaler type")
	}
}
