package cloudprovider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	log "github.com/sirupsen/logrus"
)

type azureResourceCleaner struct {
	azureClient *armresources.ResourceGroupsClient
}

type config struct {
	clientID       string
	clientSecret   string
	subscriptionID string
	tenantID       string
	userAgent      string
}

func NewAzureResourcesCleaner(secretData map[string][]byte) (ResourceCleaner, error) {
	config, err := toConfig(secretData)
	if err != nil {
		return nil, err
	}

	azureClient, err := newResourceGroupsClient(config)
	if err != nil {
		return nil, err
	}

	return &azureResourceCleaner{
		azureClient: azureClient,
	}, nil
}

func (ac azureResourceCleaner) Do() error {
	ctx := context.Background()
	pager := ac.azureClient.NewListPager(nil)

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		if nextResult.ResourceGroupListResult.Value != nil {

		}

		for _, resourceGroup := range nextResult.Value {
			if resourceGroup.Name != nil {
				log.Infof("Deleting resource group '%s'", *resourceGroup.Name)
				future, err := ac.azureClient.BeginDelete(ctx, *resourceGroup.Name, nil)
				if err != nil {
					log.Errorf("failed to init resource group '%s' deletion", *resourceGroup.Name)
					continue
				}

				_, err = future.PollUntilDone(ctx, nil)
				if err != nil {
					log.Errorf("failed to remove resource group '%s', %s: ", *resourceGroup.Name, err.Error())
				}
			}
		}
	}

	log.Info("Azure resources cleanup finished successfully!")

	return nil
}

func toConfig(secretData map[string][]byte) (config, error) {
	clientID, exists := secretData["clientID"]
	if !exists {
		return config{}, fmt.Errorf("clientID not provided in the secret")
	}

	clientSecret, exists := secretData["clientSecret"]
	if !exists {
		return config{}, fmt.Errorf("clientSecret not provided in the secret")
	}

	subscriptionID, exists := secretData["subscriptionID"]
	if !exists {
		return config{}, fmt.Errorf("subscriptionID not provided in the secret")
	}

	tenantID, exists := secretData["tenantID"]
	if !exists {
		return config{}, fmt.Errorf("tenantID not provided in the secret")
	}

	return config{
		clientID:       string(clientID),
		clientSecret:   string(clientSecret),
		subscriptionID: string(subscriptionID),
		tenantID:       string(tenantID),
		userAgent:      "kyma-environment-broker",
	}, nil
}

func newResourceGroupsClient(config config) (*armresources.ResourceGroupsClient, error) {
	credential, err := azidentity.NewClientSecretCredential(config.tenantID, config.clientID, config.clientSecret, nil)
	if err != nil {
		return nil, err
	}

	return armresources.NewResourceGroupsClient(config.subscriptionID, credential, nil)
}

// getGroupsClient gets a client for handling of Azure ResourceGroups
// func getGroupsClient(config *config, authorizer autorest.Authorizer) (resources.GroupsClient, error) {
// 	client := resources.NewGroupsClient(config.subscriptionID)
// 	client.Authorizer = authorizer

// 	if err := client.AddToUserAgent(config.userAgent); err != nil {
// 		return resources.GroupsClient{}, fmt.Errorf("while adding user agent [%s]: %w", config.userAgent, err)
// 	}

// 	return client, nil
// }

// func getResourceManagementAuthorizer(config *config, environment *azure.Environment) (autorest.Authorizer, error) {
// 	armAuthorizer, err := getAuthorizerForResource(config, environment)
// 	if err != nil {
// 		return nil, fmt.Errorf("while creating resource authorizer: %w", err)
// 	}

// 	return armAuthorizer, err
// }

// func getAuthorizerForResource(config *config, environment *azure.Environment) (autorest.Authorizer, error) {
// 	oauthConfig, err := adal.NewOAuthConfig(environment.ActiveDirectoryEndpoint, config.tenantID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.clientID, config.clientSecret, environment.ResourceManagerEndpoint)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return autorest.NewBearerAuthorizer(token), err
// }
