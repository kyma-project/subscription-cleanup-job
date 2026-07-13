# Azure Government Market Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `UnitedStatesMarket` ("united-states") to the market model and wire it up to `cloud.AzureGovernment` in the Azure SDK client options, mirroring how PR #127 added Chinese market support.

**Architecture:** The `Market` type in `model/types.go` is threaded through the entire call stack. `GetClientSecretCredentialAndClientOptions` in `azure.go` is the single place that maps a `Market` to Azure SDK cloud options. Refactor that function to use a `switch` + private helper to eliminate duplication between China and US Government.

**Tech Stack:** Go, `github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud`, `github.com/Azure/azure-sdk-for-go/sdk/azidentity`, `github.com/Azure/azure-sdk-for-go/sdk/azcore/arm`

---

### Task 1: Update the Market model

**Files:**
- Modify: `cmd/subscriptioncleanup/model/types.go`

- [ ] **Step 1: Write the failing test**

In `cmd/subscriptioncleanup/model/types.go` there are no dedicated tests for `Market`, so we verify via compilation. But first confirm the existing constant to replace:

```bash
grep -n "USGovMarket\|UnitedStatesMarket" cmd/subscriptioncleanup/model/types.go
```

Expected output:
```
32:	USGovMarket   Market = "ns2"
```

- [ ] **Step 2: Replace `USGovMarket` with `UnitedStatesMarket`**

Open `cmd/subscriptioncleanup/model/types.go`. The current `Market` block looks like:

```go
type Market string

const (
	GlobalMarket  Market = "global"
	ChineseMarket Market = "chinese"
	USGovMarket   Market = "ns2"
)

func (e Market) IsValid() bool {
	switch e {
	case GlobalMarket, ChineseMarket, USGovMarket:
		return true
	}
	return false
}

func (e Market) String() string { return string(e) }
```

Replace it with:

```go
type Market string

const (
	GlobalMarket        Market = "global"
	ChineseMarket       Market = "chinese"
	UnitedStatesMarket  Market = "united-states"
)

func (e Market) IsValid() bool {
	switch e {
	case GlobalMarket, ChineseMarket, UnitedStatesMarket:
		return true
	}
	return false
}

func (e Market) String() string { return string(e) }
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/subscriptioncleanup/model/types.go
git commit -m "feat: replace USGovMarket with UnitedStatesMarket (united-states)"
```

---

### Task 2: Refactor `GetClientSecretCredentialAndClientOptions` and add US Government support

**Files:**
- Modify: `cmd/subscriptioncleanup/cloudprovider/azure.go`
- Modify: `cmd/subscriptioncleanup/cloudprovider/azure_test.go`

- [ ] **Step 1: Write the failing tests**

Open `cmd/subscriptioncleanup/cloudprovider/azure_test.go`. The current `TestGetClientSecretCredentialOptions` test ends at line 53. Add these new test cases (and extend `TestNewAzureResourcesCleaner_WithMarket`) so the file looks like:

```go
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

	markets := []model.Market{model.GlobalMarket, model.ChineseMarket, model.UnitedStatesMarket}
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
	optsCredsGlobal, optsClientGlobal := GetClientSecretCredentialAndClientOptions(model.GlobalMarket)
	assert.Nil(t, optsCredsGlobal)
	assert.Nil(t, optsClientGlobal)

	// Chinese market should return options structs configured for Azure China
	optsCredChina, optsCliChina := GetClientSecretCredentialAndClientOptions(model.ChineseMarket)
	if assert.NotNil(t, optsCredChina) {
		assert.Equal(t, cloud.AzureChina, optsCredChina.ClientOptions.Cloud)
		assert.True(t, optsCredChina.DisableInstanceDiscovery)
	}
	if assert.NotNil(t, optsCliChina) {
		assert.Equal(t, cloud.AzureChina, optsCliChina.ClientOptions.Cloud)
	}

	// US Government market should return options structs configured for Azure Government
	optsCredUS, optsCliUS := GetClientSecretCredentialAndClientOptions(model.UnitedStatesMarket)
	if assert.NotNil(t, optsCredUS) {
		assert.Equal(t, cloud.AzureGovernment, optsCredUS.ClientOptions.Cloud)
		assert.True(t, optsCredUS.DisableInstanceDiscovery)
	}
	if assert.NotNil(t, optsCliUS) {
		assert.Equal(t, cloud.AzureGovernment, optsCliUS.ClientOptions.Cloud)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/subscriptioncleanup/cloudprovider/... -run TestGetClientSecretCredentialOptions -v
```

Expected: `FAIL` — `UnitedStatesMarket` undefined (compile error from Task 1 not yet in place, or the switch not returning Government options yet).

> Note: If Task 1 is already done, the compile error won't appear but `TestGetClientSecretCredentialOptions` will fail because `UnitedStatesMarket` falls through to `nil` — the test asserting `NotNil` will fail.

- [ ] **Step 3: Refactor `GetClientSecretCredentialAndClientOptions` in `azure.go`**

Replace the existing `GetClientSecretCredentialAndClientOptions` function (currently lines 116–131) with the switch-based version plus a private helper. The full updated bottom of `azure.go` should look like:

```go
func NewResourceGroupsClient(config config, market model.Market) (*armresources.ResourceGroupsClient, error) {
	credentialOptions, clientOptions := GetClientSecretCredentialAndClientOptions(market)

	credential, err := azidentity.NewClientSecretCredential(config.tenantID, config.clientID, config.clientSecret, credentialOptions)
	if err != nil {
		return nil, err
	}

	return armresources.NewResourceGroupsClient(config.subscriptionID, credential, clientOptions)
}

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

- [ ] **Step 4: Run all azure tests**

```bash
go test ./cmd/subscriptioncleanup/cloudprovider/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/subscriptioncleanup/cloudprovider/azure.go cmd/subscriptioncleanup/cloudprovider/azure_test.go
git commit -m "feat: add AzureGovernment client options for UnitedStatesMarket, refactor to switch"
```

---

### Task 3: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update `APP_MARKET` description**

In `README.md` line 21, the current text is:

```
   - **APP_MARKET** - required for some restricted markets patching, such as specific Azure client creation for Chinese markets (optional, possible values: `global`/`chinese`, default: `global`)
```

Replace it with:

```
   - **APP_MARKET** - required for some restricted markets patching, such as specific Azure client creation for sovereign clouds (optional, possible values: `global`/`chinese`/`united-states`, default: `global`)
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add united-states as valid APP_MARKET value"
```

---

### Task 4: Run full test suite

- [ ] **Step 1: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS, no compilation errors.
