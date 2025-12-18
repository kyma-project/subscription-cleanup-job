package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/model"
	"github.com/stretchr/testify/assert"
)

func TestNewAzureResourcesCleaner_MissingSecrets(t *testing.T) {
	_, err := NewAzureResourcesCleaner(map[string][]byte{}, model.GlobalMarket)
	assert.Error(t, err)
}

func TestNewAzureResourcesCleaner_WithMarket(t *testing.T) {
	secretData := map[string][]byte{
		"clientID":       []byte("client-id"),
		"clientSecret":   []byte("client-secret"),
		"subscriptionID": []byte("sub-id"),
		"tenantID":       []byte("tenant-id"),
	}

	markets := []model.Market{model.GlobalMarket, model.ChineseMarket}
	for _, m := range markets {
		m := m
		t.Run(m.String(), func(t *testing.T) {
			rc, err := NewAzureResourcesCleaner(secretData, m)
			assert.NoError(t, err)
			assert.NotNil(t, rc)

			ac, _ := rc.(*azureResourceCleaner)
			assert.NotNil(t, ac.azureClient)
		})
	}
}

func TestGetClientSecretCredentialOptions(t *testing.T) {
	// Global market should return nil options (use default Azure cloud)
	optsGlobal := GetClientSecretCredentialOptions(model.GlobalMarket)
	assert.Nil(t, optsGlobal)

	// Chinese market should return an options struct configured for Azure China
	optsChina := GetClientSecretCredentialOptions(model.ChineseMarket)
	if assert.NotNil(t, optsChina) {
		assert.Equal(t, cloud.AzureChina, optsChina.ClientOptions.Cloud)
	}
}
