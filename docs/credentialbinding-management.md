# CredentialBinding Management: KEB ↔ KIM ↔ SCJ

```mermaid
sequenceDiagram
    autonumber
    participant Admin as Admin / Migration Tool
    participant GardenerAPI as Gardener API<br/>(kcp-system)
    participant KEB as KEB<br/>ResolveCredentialsBindingStep
    participant KEBDB as KEB DB<br/>(PostgreSQL)
    participant KIMCtrl as KIM<br/>Runtime FSM
    participant SCJ as SCJ<br/>CredentialBindingCleaner
    participant Cloud as Cloud Provider<br/>(AWS / Azure / GCP)

    Note over Admin,GardenerAPI: Pool Initialisation (one-time / migration)
    Admin->>GardenerAPI: Create CredentialsBinding<br/>labels: hyperscalerType, [shared], [internal], [euAccess]
    Note over GardenerAPI: State: UNASSIGNED<br/>(no tenantName label)

    Note over KEB,KEBDB: Provisioning — claim a binding
    KEB->>GardenerAPI: List CredentialsBindings<br/>(filter by hyperscalerType, plan, region)
    GardenerAPI-->>KEB: Available (unassigned) bindings
    KEB->>GardenerAPI: Update: set label tenantName = GlobalAccountID
    Note over GardenerAPI: State: CLAIMED
    KEB->>KEBDB: Store SubscriptionSecretName<br/>+ TargetSecret in instance row

    Note over KIMCtrl: Shoot creation — reference the binding
    KEB->>GardenerAPI: Create Runtime CR<br/>spec.shoot.secretBindingName = <binding>
    KIMCtrl->>GardenerAPI: Watch Runtime CR
    KIMCtrl->>GardenerAPI: Create / Patch Shoot<br/>spec.credentialsBindingName = <binding><br/>(clears spec.secretBindingName if EnableCredentialBinding=true)
    Note over GardenerAPI: Shoot uses CredentialsBinding<br/>to authenticate to cloud provider

    Note over KEB,GardenerAPI: Deprovisioning — mark binding dirty
    KEB->>GardenerAPI: List Shoots referencing this binding
    alt no Shoots reference it AND binding is not shared/internal
        KEB->>GardenerAPI: Update: set label dirty = "true"
        Note over GardenerAPI: State: DIRTY
    else still in use or shared/internal
        KEB-->>KEB: Skip — binding stays CLAIMED
    end

    Note over SCJ,Cloud: Cleanup Job (periodic) — release binding
    SCJ->>GardenerAPI: List CredentialsBindings where dirty = "true"
    GardenerAPI-->>SCJ: Dirty bindings
    SCJ->>GardenerAPI: Verify no Shoots reference binding
    SCJ->>GardenerAPI: Read bound Secret (cloud credentials)
    SCJ->>Cloud: Provider-specific cleanup<br/>(release quotas / deregister resources)
    Cloud-->>SCJ: OK
    SCJ->>GardenerAPI: Update: remove dirty + tenantName labels
    Note over GardenerAPI: State: UNASSIGNED<br/>(back in pool, ready for reuse)
```

---

## CredentialBinding Label State Machine

```mermaid
stateDiagram-v2
    [*] --> UNASSIGNED : Admin creates / SCJ releases

    UNASSIGNED --> CLAIMED : KEB sets tenantName = GlobalAccountID\n(ResolveCredentialsBindingStep)

    CLAIMED --> DIRTY : KEB sets dirty = true\n(FreeCredentialsBindingStep)\n[no Shoots reference it AND not shared/internal]

    CLAIMED --> CLAIMED : Skip — binding still in use\nor shared / internal

    DIRTY --> UNASSIGNED : SCJ removes dirty + tenantName labels\nafter cloud-provider resource cleanup

    note right of UNASSIGNED
        Labels: hyperscalerType, [shared], [internal], [euAccess]
        No tenantName, no dirty
    end note

    note right of CLAIMED
        + tenantName = GlobalAccountID
        Shoot.spec.credentialsBindingName references this binding
    end note

    note right of DIRTY
        + dirty = "true"
        tenantName still present
        Awaiting SCJ cleanup run
    end note
```

---

## Cluster Access Map

Which Kubernetes clusters each component talks to, and why.

```mermaid
graph TD
    subgraph KCP["KCP cluster (in-cluster)"]
        KEB["KEB<br/>(Kyma Environment Broker)"]
        KIM["KIM<br/>(Kyma Infrastructure Manager)"]
        KCPSECRETS["kubeconfig-&lt;runtimeID&gt; Secrets<br/>(kcp-system)"]
        KCPCRS["Runtime / Kyma CRs<br/>(kcp-system)"]
    end

    subgraph Gardener["Gardener cluster"]
        SHOOTS["Shoots"]
        CREDBINDINGS["CredentialsBindings / SecretBindings"]
        CLOUDSECRETS["Cloud-credential Secrets"]
    end

    subgraph SKR["SKR (Shoot Kubernetes Runtime)"]
        SKRRESOURCES["kyma-system resources<br/>(ClusterRoleBindings, namespaces, …)"]
    end

    SCJ["SCJ<br/>(Subscription Cleanup Job)<br/>one-shot pod on KCP"]

    KEB -->|"in-cluster REST config<br/>(reads/writes Runtime CRs, kubeconfig secrets)"| KCPCRS
    KEB -->|"Gardener kubeconfig<br/>(list/label CredentialsBindings & Shoots)"| CREDBINDINGS
    KEB -->|"Gardener kubeconfig<br/>(list Shoots, read cloud-cred Secrets)"| SHOOTS
    KEB -->|"kubeconfig from KCP Secret<br/>(provision/deprovision SKR resources)"| SKRRESOURCES

    KIM -->|"in-cluster REST config<br/>(watches Runtime CRs, stores kubeconfig secrets)"| KCPCRS
    KIM -->|"in-cluster REST config<br/>(reads kubeconfig-&lt;runtimeID&gt; to reach SKR)"| KCPSECRETS
    KIM -->|"Gardener kubeconfig<br/>(create/patch Shoots)"| SHOOTS
    KIM -->|"kubeconfig from KCP Secret<br/>(bootstrap SKR: namespaces, CRBs, audit-log creds)"| SKRRESOURCES

    SCJ -->|"Gardener kubeconfig<br/>(list dirty CredentialsBindings, read cloud-cred Secrets)"| CREDBINDINGS
    SCJ -->|"Gardener kubeconfig<br/>(verify no Shoots reference binding)"| SHOOTS
    SCJ -->|"Gardener kubeconfig<br/>(read bound cloud-credential Secret)"| CLOUDSECRETS
```

| Component | KCP cluster | Gardener cluster | SKR (shoot) |
|-----------|:-----------:|:----------------:|:-----------:|
| KEB | ✅ in-cluster | ✅ via kubeconfig | ✅ via KCP Secret |
| KIM | ✅ in-cluster | ✅ via kubeconfig | ✅ via KCP Secret |
| SCJ | ❌ | ✅ via kubeconfig | ❌ |

**KCP** = Kyma Control Plane (hosts Runtime/Kyma CRs and kubeconfig Secrets for each SKR)  
**Gardener** = Gardener project namespace (hosts Shoots, CredentialsBindings, cloud-cred Secrets)  
**SKR** = Shoot Kubernetes Runtime (the end-user cluster provisioned by Gardener)

---

## Component Responsibility Summary

| Action | Component | Mechanism |
|--------|-----------|-----------|
| Create CredentialsBinding in pool | Admin / Migration tool | `hack/credentialbindings/main.go` |
| Claim binding (set `tenantName`) | KEB — `ResolveCredentialsBindingStep` | `UpdateCredentialsBinding()` via Gardener client |
| Store assignment | KEB — DB write | `SubscriptionSecretName` in instance row |
| Reference binding in Shoot | KIM — Runtime FSM | `ExtendWithCredentialsBinding()` → `Shoot.spec.credentialsBindingName` |
| Mark dirty (`dirty=true`) | KEB — `FreeCredentialsBindingStep` | Label update on deprovisioning |
| Cloud-resource cleanup + label reset | SCJ — `CredentialBindingCleaner.Do()` | Provider cleaner → remove `dirty` + `tenantName` |

**SCJ** = Subscription Cleanup Job (`subscription-cleanup-job`)
**KIM** = Kyma Infrastructure Manager (`kyma-infrastructure-manager`)
**KEB** = Kyma Environment Broker (`kyma-environment-broker`)
