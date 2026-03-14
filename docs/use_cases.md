# Use Cases Overview — IDP Test Environment Platform

**Source:** [use_cases.puml](use_cases.puml) | **Requirements:** [requirements.md](requirements.md)

---

```mermaid
flowchart LR
    dev["Developer"]
    qa["QA Engineer"]
    lead["Team Lead"]
    pe["Platform Engineer"]

    subgraph EM["Environment Management"]
        UC001("UC-001\nSubmit environment manifest")
        UC002("UC-002\nEnable application component")
        UC003("UC-003\nDisable application component")
        UC004("UC-004\nProvision database alongside app")
        UC005("UC-005\nCreate environment via portal")
        UC006("UC-006\nBrowse environment catalog")
        UC007("UC-007\nView environment status")
        UC008("UC-008\nDelete environment")
        UC011("UC-011\nIsolate environment per namespace")
        UC012("UC-012\nOverride component configuration")
        UC014("UC-014\nView provisioning logs")
    end

    subgraph PA["Platform Administration"]
        UC009("UC-009\nDefine environment template")
        UC010("UC-010\nRegister new component type")
        UC013("UC-013\nConfigure GitOps reconciliation")
    end

    subgraph MIG["Migration from Legacy Platform"]
        UC015("UC-015\nConvert docker-compose to IDP manifest")
        UC016("UC-016\nExport legacy environment config")
        UC017("UC-017\nValidate migrated environment")
        UC018("UC-018\nFollow migration runbook")
        UC019("UC-019\nTrack migration status per team")
    end

    dev --> UC001
    dev --> UC002
    dev --> UC003
    dev --> UC004
    dev --> UC005
    dev --> UC006
    dev --> UC007
    dev --> UC008
    dev --> UC012
    dev --> UC014
    dev --> UC015
    dev --> UC017
    dev --> UC018

    qa --> UC006
    qa --> UC007
    qa --> UC011
    qa --> UC017

    lead --> UC009
    lead --> UC006
    lead --> UC007

    pe --> UC010
    pe --> UC013
    pe --> UC016
    pe --> UC019

    UC002 -.->|"«include»"| UC004
    UC005 -.->|"«include»"| UC001
```

---

## Traceability

| UC      | Title                               | FR      |
|---------|-------------------------------------|---------|
| UC-001  | Submit environment manifest         | FR-001  |
| UC-002  | Enable application component        | FR-002  |
| UC-003  | Disable application component       | FR-003  |
| UC-004  | Provision database alongside app    | FR-004  |
| UC-005  | Create environment via portal       | FR-005  |
| UC-006  | Browse environment catalog          | FR-006  |
| UC-007  | View environment status             | FR-007  |
| UC-008  | Delete environment                  | FR-008  |
| UC-009  | Define environment template         | FR-009  |
| UC-010  | Register new component type         | FR-010  |
| UC-011  | Isolate environment per namespace   | FR-011  |
| UC-012  | Override component configuration    | FR-012  |
| UC-013  | Configure GitOps reconciliation     | FR-013  |
| UC-014  | View provisioning logs              | FR-014  |
| UC-015  | Convert docker-compose to IDP manifest | FR-015 |
| UC-016  | Export legacy environment config    | FR-016  |
| UC-017  | Validate migrated environment       | FR-017  |
| UC-018  | Follow migration runbook            | FR-018  |
| UC-019  | Track migration status per team     | FR-019  |
