package job

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/cloudprovider/mocks"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	credentialBindingGVK = schema.GroupVersionKind{Group: "security.gardener.cloud", Version: "v1alpha1", Kind: "CredentialsBinding"}
)

func TestCredentialCleanerJob(t *testing.T) {
	t.Run("should return credential binding to the credentials pool", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "credential-secret1", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}
		credentialBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding1",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "Secret",
					"name":      "credential-secret1",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "azure",
				},
			},
		}
		credentialBinding.SetGroupVersionKind(credentialBindingGVK)

		mockClient := fake.NewClientset(secret)

		gardenerFake := gardener.NewDynamicFakeClient(credentialBinding)
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedCredentialBinding, err := mockCredentialBindings.Get(context.Background(), credentialBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "", cleanedCredentialBinding.GetLabels()["dirty"])
		assert.Equal(t, "", cleanedCredentialBinding.GetLabels()["tenantName"])
	})

	t.Run("should not return credential binding to the credentials pool when credential is still in use", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "credential-secret1", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}
		credentialBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding1",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "Secret",
					"name":      "credential-secret1",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "azure",
				},
			},
		}
		credentialBinding.SetGroupVersionKind(credentialBindingGVK)

		shoot := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "some-shoot",
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"credentialsBindingName": credentialBinding.GetName(),
				},
				"status": map[string]interface{}{},
			},
		}
		shoot.SetGroupVersionKind(shootGVK)

		mockClient := fake.NewClientset(secret)

		gardenerFake := gardener.NewDynamicFakeClient(credentialBinding, shoot)
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedCredentialBinding, err := mockCredentialBindings.Get(context.Background(), credentialBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "true", cleanedCredentialBinding.GetLabels()["dirty"])
		assert.Equal(t, "tenant1", cleanedCredentialBinding.GetLabels()["tenantName"])
	})

	t.Run("should handle multiple credential bindings correctly", func(t *testing.T) {
		//given
		secret1 := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "credential-secret1", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}
		secret2 := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "credential-secret2", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret2"),
				"clientID":       []byte("tenant2"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12345"),
				"tenantID":       []byte("tenant2"),
			},
		}

		credentialBinding1 := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding1",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "Secret",
					"name":      "credential-secret1",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "azure",
				},
			},
		}
		credentialBinding1.SetGroupVersionKind(credentialBindingGVK)

		credentialBinding2 := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding2",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant2",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "Secret",
					"name":      "credential-secret2",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "azure",
				},
			},
		}
		credentialBinding2.SetGroupVersionKind(credentialBindingGVK)

		// Shoot only references credentialBinding2, so credentialBinding1 should be released
		shoot := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "some-shoot",
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"credentialsBindingName": credentialBinding2.GetName(),
				},
				"status": map[string]interface{}{},
			},
		}
		shoot.SetGroupVersionKind(shootGVK)
		mockClient := fake.NewClientset(secret1, secret2)

		gardenerFake := gardener.NewDynamicFakeClient(credentialBinding1, credentialBinding2, shoot)
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)

		// credentialBinding1 should be released
		cleanedCredentialBinding1, err := mockCredentialBindings.Get(context.Background(), credentialBinding1.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "", cleanedCredentialBinding1.GetLabels()["dirty"])
		assert.Equal(t, "", cleanedCredentialBinding1.GetLabels()["tenantName"])

		// credentialBinding2 should still be marked as dirty
		cleanedCredentialBinding2, err := mockCredentialBindings.Get(context.Background(), credentialBinding2.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "true", cleanedCredentialBinding2.GetLabels()["dirty"])
		assert.Equal(t, "tenant2", cleanedCredentialBinding2.GetLabels()["tenantName"])
	})

	t.Run("should handle credential binding with different hyperscaler types", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "credential-secret-gcp", Namespace: namespace,
			},
			Data: map[string][]byte{
				"serviceAccountKey": []byte("gcp-key"),
			},
		}
		credentialBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding-gcp",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant-gcp",
						"hyperscalerType": "gcp",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "Secret",
					"name":      "credential-secret-gcp",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "gcp",
				},
			},
		}
		credentialBinding.SetGroupVersionKind(credentialBindingGVK)

		mockClient := fake.NewClientset(secret)

		gardenerFake := gardener.NewDynamicFakeClient(credentialBinding)
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.GCP, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedCredentialBinding, err := mockCredentialBindings.Get(context.Background(), credentialBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "", cleanedCredentialBinding.GetLabels()["dirty"])
		assert.Equal(t, "", cleanedCredentialBinding.GetLabels()["tenantName"])
		providerFactory.AssertCalled(t, "New", model.GCP, mock.Anything)
	})

	t.Run("should not fail when no dirty credential bindings exist", func(t *testing.T) {
		//given
		mockClient := fake.NewClientset()

		gardenerFake := gardener.NewDynamicFakeClient()
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		providerFactory := &mocks.ProviderFactory{}

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		providerFactory.AssertNotCalled(t, "New")
	})

	t.Run("should skip credential binding when credentialsRef is not a Secret", func(t *testing.T) {
		//given
		credentialBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "credentialBinding-configmap",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"credentialsRef": map[string]interface{}{
					"kind":      "ConfigMap",
					"name":      "some-configmap",
					"namespace": namespace,
				},
				"provider": map[string]interface{}{
					"type": "azure",
				},
			},
		}
		credentialBinding.SetGroupVersionKind(credentialBindingGVK)

		mockClient := fake.NewClientset()

		gardenerFake := gardener.NewDynamicFakeClient(credentialBinding)
		mockCredentialBindings := gardenerFake.Resource(gardener.CredentialsBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		providerFactory := &mocks.ProviderFactory{}

		cleaner := NewCredentialBindingCleaner(context.Background(), mockClient, mockCredentialBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)

		// Credential binding should still be marked as dirty because it couldn't be processed
		cleanedCredentialBinding, err := mockCredentialBindings.Get(context.Background(), credentialBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "true", cleanedCredentialBinding.GetLabels()["dirty"])
	})
}
