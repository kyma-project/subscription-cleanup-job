[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/subscription-cleanup-job)](https://api.reuse.software/info/github.com/kyma-project/subscription-cleanup-job)

# Kyma Control Plane - Subscription Cleanup Job

## Overview

Subscription Cleanup Job is a subcomponent of [Kyma Control Plane](https://github.com/kyma-project/control-plane) for the cleanup of managed subscriptions from [Hyperscaler Account Pool (HAP)](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-10-hyperscaler-account-pool.md).

For more information on KCP and its components, read the [KCP documentation](https://github.com/kyma-project/control-plane/tree/main/docs) and the [KEB documentation](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs).

## Runnining locally

- download your Gardener Kubeconfig (personal kubeconfig, service account isn't needed)
- build subscription cleanup job: `go build ./cmd/subscriptioncleanup/main.go`
- set environmental variables:
  - `APP_GARDENER_PROJECT=frog-dev`
  - `APP_GARDENER_KUBECONFIG_PATH=$PWD/kubeconfig-garden-frog-dev.yaml`
- run the job: `./main`

SCJ will look for secret bindings in the Gardener with `dirty=true` and `hyperscalerType` labels.
If `hyperscalerType` is not set, the secret will be logged and ignored.
If `dirty=true` is not set, the secret will be ignored.
If there are shoots assigned to the secret bindings, it will be logged and ignored.

> [!CAUTION]
> SCJ will remove all resources associated with the secret bindings.
> This will break any existing shoots associated with the subscription.
>
> Make sure nobody is using the subscription (can be a different secret) before running SCJ.

Example correct run with nothing to clean:
```
$ APP_GARDENER_PROJECT=frog-dev APP_GARDENER_KUBECONFIG_PATH=$PWD/kubeconfig-garden-frog-dev.yaml ./main
INFO[0000] Starting cleanup job!
INFO[0000] Started releasing resources
Please visit the following URL in your browser: http://localhost:8000
INFO[0020] Finished releasing resources
INFO[0020] # HALT ISTIO SIDECAR #
ERRO[0020] unable to send post request to quit Istio sidecar: Post "http://127.0.0.1:15020/quitquitquit": dial tcp 127.0.0.1:15020: connect: connection refused
INFO[0020] Cleanup job finished successfully!
```

## Contributing

See the [Contributing Rules](CONTRIBUTING.md).

## Code of Conduct

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing

See the [license](./LICENSE) file.
