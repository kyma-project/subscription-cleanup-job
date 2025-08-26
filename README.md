[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/subscription-cleanup-job)](https://api.reuse.software/info/github.com/kyma-project/subscription-cleanup-job)
[![unit tests](https://github.com/kyma-project/subscription-cleanup-job/actions/workflows/unit-tests.yaml/badge.svg)](https://github.com/kyma-project/subscription-cleanup-job/actions/workflows/unit-tests.yaml)
[![Coverage Status](https://coveralls.io/repos/github/kyma-project/subscription-cleanup-job/badge.svg)](https://coveralls.io/github/kyma-project/subscription-cleanup-job)

# Kyma Control Plane - Subscription Cleanup Job

## Overview

Subscription Cleanup Job (SCJ) is a subcomponent of [Kyma Control Plane](https://github.com/kyma-project/control-plane) for the cleanup of managed subscriptions from [Hyperscaler Account Pool (HAP)](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-10-hyperscaler-account-pool.md).

For more information on KCP and its components, read the [KCP documentation](https://github.com/kyma-project/control-plane/tree/main/docs) and the [KEB documentation](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs).

## Run SCJ Locally

1. Download your Gardener kubeconfig. It can be a personal kubeconfig; a service account isn't required.
2. To build an SCJ, run: `go build ./cmd/subscriptioncleanup/main.go`.
3. Set the environmental variables:
   - `APP_GARDENER_PROJECT=frog-dev`
   - `APP_GARDENER_KUBECONFIG_PATH=$PWD/kubeconfig-garden-frog-dev.yaml`
4. Run the job: `./main`.

SCJ searches the Gardener instance for secret bindings with `dirty=true` and `hyperscalerType` labels.
If `hyperscalerType` is not set, the Secret is logged and ignored.
If `dirty=true` is not set, the Secret is ignored.
If there are shoots assigned to the secret bindings, the Secret is logged and ignored.

> [!CAUTION]
> SCJ removes all resources associated with the secret bindings.
> This causes breaking any existing shoots associated with the subscription.
>
> Before running SCJ, ensure nobody is using the subscription and multiple Secrets are not assigned to the subscription.

See an example of a correct run with no dirty Secrets:

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

See an example of a correct run after `k label secretbindings/gardener-secret dirty=true` with no resources to delete:
```
$ APP_GARDENER_PROJECT=frog-dev APP_GARDENER_KUBECONFIG_PATH=$PWD/kubeconfig-garden-frog-dev.yaml ./main
INFO[0000] Starting cleanup job!
INFO[0000] Started releasing resources
INFO[0000] Checking gardener-secret
INFO[0000] Switching to region af-south-1
INFO[0001] Switching to region ap-south-1
INFO[0002] Switching to region eu-north-1
INFO[0002] Switching to region eu-west-3
INFO[0003] Switching to region eu-south-1
INFO[0003] Switching to region eu-west-2
INFO[0003] Switching to region eu-west-1
INFO[0004] Switching to region ap-northeast-3
INFO[0005] Switching to region ap-northeast-2
INFO[0007] Switching to region me-south-1
INFO[0007] Switching to region ap-northeast-1
INFO[0009] Switching to region ca-central-1
INFO[0010] Switching to region sa-east-1
INFO[0011] Switching to region ap-east-1
INFO[0012] Switching to region ap-southeast-1
INFO[0014] Switching to region ap-southeast-2
INFO[0015] Switching to region eu-central-1
INFO[0015] Switching to region us-east-1
INFO[0016] Switching to region us-east-2
INFO[0017] Switching to region us-west-1
INFO[0018] Switching to region us-west-2
INFO[0019] Resources released for 'gardener-secret' secret binding
INFO[0019] Finished releasing resources
INFO[0019] # HALT ISTIO SIDECAR #
ERRO[0019] unable to send post request to quit Istio sidecar: Post "http://127.0.0.1:15020/quitquitquit": dial tcp 127.0.0.1:15020: connect: connection refused
INFO[0019] Cleanup job finished successfully!
```

If any resources are deleted, SCJ logs them.

## Contributing

See the [Contributing Rules](CONTRIBUTING.md).

## Code of Conduct

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing

See the [license](./LICENSE) file.
