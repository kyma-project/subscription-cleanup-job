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

Example correct run with no dirty secrets:
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

Example correct run after `k label secretbindings/gardener-secret dirty=true` (no resources to delete):
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
