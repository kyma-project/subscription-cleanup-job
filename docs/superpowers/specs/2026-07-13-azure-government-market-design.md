# Azure Government Market Support

**Date:** 2026-07-13

## Context

PR #127 added support for the Chinese Azure sovereign cloud by threading a `Market` value through the call stack and using it in `GetClientSecretCredentialAndClientOptions` to configure the Azure SDK with `cloud.AzureChina`. This spec describes the parallel change for the US Government (Azure Government) sovereign cloud.

## Changes

### `model/types.go`

- Remove `USGovMarket Market = "ns2"`
- Add `UnitedStatesMarket Market = "united-states"`
- Add `UnitedStatesMarket` to `IsValid()`

### `cloudprovider/azure.go`

Replace the `if market == model.ChineseMarket` block with a `switch` and extract a private `cloudOptions` helper to eliminate duplication:

```go
func GetClientSecretCredentialAndClientOptions(market model.Market) (*azidentity.ClientSecretCredentialOptions, *arm.ClientOptions) {
    switch market {
    case model.ChineseMarket:
        return cloudOptions(cloud.AzureChina)
    case model.UnitedStatesMarket:
        return cloudOptions(cloud.AzureGovernment)
    default:
        return nil, nil
    }
}

func cloudOptions(c cloud.Configuration) (*azidentity.ClientSecretCredentialOptions, *arm.ClientOptions) {
    return &azidentity.ClientSecretCredentialOptions{
            ClientOptions:            policy.ClientOptions{Cloud: c},
            DisableInstanceDiscovery: true,
        },
        &arm.ClientOptions{
            ClientOptions: policy.ClientOptions{Cloud: c},
        }
}
```

### `cloudprovider/azure_test.go`

- Add `model.UnitedStatesMarket` to the `markets` slice in `TestNewAzureResourcesCleaner_WithMarket`
- Add assertions for `UnitedStatesMarket` in `TestGetClientSecretCredentialOptions` verifying `cloud.AzureGovernment`

### `README.md`

Update the `APP_MARKET` description to include `united-states` as a valid value alongside `global` and `chinese`.

## Out of Scope

- No changes to `cleanerSecret.go`, `cleanerCredential.go`, `main.go`, or the mock — they already accept `model.Market` generically.
