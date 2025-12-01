package job

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/cloudprovider"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/model"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func NewCredentialBindingCleaner(context context.Context,
	kubernetesInterface kubernetes.Interface,
	credentialBindingsClient dynamic.ResourceInterface,
	shootClient dynamic.ResourceInterface,
	providerFactory cloudprovider.ProviderFactory) Cleaner {

	return &credentialCleaner{
		kubernetesInterface:      kubernetesInterface,
		credentialBindingsClient: credentialBindingsClient,
		providerFactory:          providerFactory,
		shootClient:              shootClient,
		context:                  context,
	}
}

type credentialCleaner struct {
	kubernetesInterface      kubernetes.Interface
	credentialBindingsClient dynamic.ResourceInterface
	providerFactory          cloudprovider.ProviderFactory
	shootClient              dynamic.ResourceInterface
	context                  context.Context
}

func (p *credentialCleaner) Do() error {
	logrus.Infof("Started releasing credential binding resources")
	credentialBindings, err := p.getCredentialBindingsToRelease()
	if err != nil {
		return err
	}
	for _, credentialBinding := range credentialBindings {
		canRelease, err := p.checkIfCredentialCanBeReleased(credentialBinding)
		if err != nil {
			logrus.Errorf("Failed to list shoots for '%s' credential binding: %s", credentialBinding.GetName(), err.Error())
			continue
		}

		if !canRelease {
			logrus.Infof("Credential binding '%s' cannot be released yet", credentialBinding.GetName())
			continue
		}

		err = p.releaseCredentialBindingResources(credentialBinding)
		if err != nil {
			logrus.Errorf("Failed to release resources for '%s' credential binding: %s", credentialBinding.GetName(), err.Error())
			continue
		}
		err = p.returnCredentialBindingToThePool(credentialBinding.GetName())
		if err != nil {
			logrus.Errorf("Failed returning '%s' credential binding to the pool: %s", credentialBinding.GetName(), err.Error())
			continue
		}
		logrus.Infof("Resources released for '%s' credential binding", credentialBinding.GetName())
	}

	logrus.Info("Finished releasing resources")
	return nil
}

func (p *credentialCleaner) getCredentialBindingsToRelease() ([]unstructured.Unstructured, error) {
	labelSelector := fmt.Sprintf("dirty=true")

	return getCredentialBindings(p.context, p.credentialBindingsClient, labelSelector)
}

func getCredentialBindings(ctx context.Context, credentialBindingsClient dynamic.ResourceInterface, labelSelector string) ([]unstructured.Unstructured, error) {
	credentials, err := credentialBindingsClient.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing credential bindings for LabelSelector: %s: %w", labelSelector, err)
	}

	return credentials.Items, nil
}

func (p *credentialCleaner) checkIfCredentialCanBeReleased(binding unstructured.Unstructured) (bool, error) {
	list, err := p.shootClient.List(p.context, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list shoots: %w", err)
	}

	for _, sh := range list.Items {
		shoot := gardener.Shoot{
			Unstructured: sh,
		}
		if shoot.GetSpecCredentialsBindingName() == binding.GetName() {
			return false, nil
		}
	}

	return true, nil
}

func (p *credentialCleaner) releaseCredentialBindingResources(credentialBinding unstructured.Unstructured) error {
	providerType, ok, err := unstructured.NestedString(credentialBinding.Object, "provider", "type")
	if err != nil || !ok {
		return fmt.Errorf("provider.type field not found or invalid")
	}

	hyperscalerType, err := model.NewHyperscalerType(providerType)
	if err != nil {
		return err
	}

	secret, err := p.getBoundSecret(credentialBinding)
	if err != nil {
		return fmt.Errorf("getting referenced secret: %w", err)
	}

	cleaner, err := p.providerFactory.New(hyperscalerType, secret.Data)
	if err != nil {
		return fmt.Errorf("initializing cloud provider cleaner: %w", err)
	}

	return cleaner.Do()
}

func (p *credentialCleaner) getBoundSecret(binding unstructured.Unstructured) (*apiv1.Secret, error) {
	credentialBinding := gardener.CredentialsBinding{
		Unstructured: binding,
	}

	secret, err := p.kubernetesInterface.CoreV1().
		Secrets(credentialBinding.GetSecretRefNamespace()).
		Get(p.context, credentialBinding.GetSecretRefName(), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting %s/%s secret: %w", credentialBinding.GetSecretRefNamespace(),
			credentialBinding.GetSecretRefName(), err)
	}
	return secret, nil
}

func (p *credentialCleaner) returnCredentialBindingToThePool(credentialBindingName string) error {
	cb, err := p.credentialBindingsClient.Get(p.context, credentialBindingName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	labels := cb.GetLabels()
	delete(labels, "dirty")
	delete(labels, "tenantName")
	cb.SetLabels(labels)

	_, err = p.credentialBindingsClient.Update(p.context, cb, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to return credential binding to the hyperscaler account pool: %w", err)
	}

	return nil
}
