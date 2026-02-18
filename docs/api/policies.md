# 📘 Authorization Policy Cookbook

> Last updated: 2026-02-18
> Applies to: API v1

Ready-to-use policy templates for common service roles.

## 📑 Table of Contents

- [Policy structure](#policy-structure)
- [1) Read-only service](#1-read-only-service)
- [2) CI writer](#2-ci-writer)
- [3) Transit encrypt-only service](#3-transit-encrypt-only-service)
- [4) Transit decrypt-only service](#4-transit-decrypt-only-service)
- [5) Audit log reader](#5-audit-log-reader)
- [6) Break-glass admin (emergency)](#6-break-glass-admin-emergency)
- [7) Key operator](#7-key-operator)
- [8) Tokenization operator](#8-tokenization-operator)
- [Policy mismatch example (wrong vs fixed)](#policy-mismatch-example-wrong-vs-fixed)
- [Common policy mistakes](#common-policy-mistakes)
- [Best practices](#best-practices)

## Compatibility

- API surface: policies used by `/v1/clients` payloads
- Server expectation: Secrets server with capability-based authorization enabled
- OpenAPI baseline: `docs/openapi.yaml`

## Policy structure

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["read"]
  }
]
```

Capabilities: `read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`.

Endpoint capability intent (quick map, condensed from [Capability matrix](capability-matrix.md)):

| Endpoint family | Typical capability |
| --- | --- |
| `GET /v1/clients`, `GET /v1/audit-logs`, `POST /v1/tokenization/validate` | `read` |
| `POST /v1/clients`, `PUT /v1/clients/:id`, `POST /v1/transit/keys`, `POST /v1/tokenization/keys` | `write` |
| `DELETE /v1/clients/:id`, `DELETE /v1/transit/keys/:id`, `DELETE /v1/tokenization/keys/:id`, `POST /v1/tokenization/revoke` | `delete` |
| `POST /v1/secrets/*path`, `POST /v1/transit/keys/:name/encrypt`, `POST /v1/tokenization/keys/:name/tokenize` | `encrypt` |
| `GET /v1/secrets/*path`, `POST /v1/transit/keys/:name/decrypt`, `POST /v1/tokenization/detokenize` | `decrypt` |
| `POST /v1/transit/keys/:name/rotate`, `POST /v1/tokenization/keys/:name/rotate` | `rotate` |

## 1) Read-only service

Use when a service only reads existing secrets.

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["decrypt"]
  }
]
```

Risk note: cannot create/update/delete secrets.

## 2) CI writer

Use when CI/CD needs to publish/update secrets.

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["encrypt"]
  }
]
```

Risk note: should not include `decrypt` unless CI must read values.

## 3) Transit encrypt-only service

Use for services that should encrypt sensitive values but never decrypt.

See [Transit API](transit.md) for encrypt/decrypt request and response contracts.

```json
[
  {
    "path": "/v1/transit/keys/payment/encrypt",
    "capabilities": ["encrypt"]
  }
]
```

Risk note: encrypt-only separation limits plaintext exposure.

## 4) Transit decrypt-only service

Use for tightly scoped decryption workers.

See [Decrypt input contract](transit.md#decrypt-input-contract) for required
`ciphertext` format.

```json
[
  {
    "path": "/v1/transit/keys/payment/decrypt",
    "capabilities": ["decrypt"]
  }
]
```

Risk note: protect runtime and logs because plaintext is handled here.

## 5) Audit log reader

Use for monitoring/compliance pipelines.

```json
[
  {
    "path": "/v1/audit-logs",
    "capabilities": ["read"]
  }
]
```

Risk note: may expose sensitive metadata (IP/user-agent/path). Restrict access.

## 6) Break-glass admin (emergency)

Use only for controlled emergency procedures.

```json
[
  {
    "path": "*",
    "capabilities": ["read", "write", "delete", "encrypt", "decrypt", "rotate"]
  }
]
```

Risk note: maximum privilege. Require approvals, short validity, and strong audit review.

## 7) Key operator

Use for teams responsible only for transit key lifecycle.

```json
[
  {
    "path": "/v1/transit/keys",
    "capabilities": ["write"]
  },
  {
    "path": "/v1/transit/keys/*/rotate",
    "capabilities": ["rotate"]
  },
  {
    "path": "/v1/transit/keys/*",
    "capabilities": ["delete"]
  }
]
```

Risk note: scope key names by environment when possible (for example `/v1/transit/keys/prod-*`).

## 8) Tokenization operator

Use for services that manage tokenization keys and token lifecycle operations.

```json
[
  {
    "path": "/v1/tokenization/keys",
    "capabilities": ["write"]
  },
  {
    "path": "/v1/tokenization/keys/*/rotate",
    "capabilities": ["rotate"]
  },
  {
    "path": "/v1/tokenization/keys/*/tokenize",
    "capabilities": ["encrypt"]
  },
  {
    "path": "/v1/tokenization/detokenize",
    "capabilities": ["decrypt"]
  },
  {
    "path": "/v1/tokenization/validate",
    "capabilities": ["read"]
  },
  {
    "path": "/v1/tokenization/revoke",
    "capabilities": ["delete"]
  }
]
```

Risk note: avoid wildcard tokenization access for application clients that only need tokenize or detokenize.

## Policy mismatch example (wrong vs fixed)

Wrong policy (insufficient capability for secret reads):

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["read"]
  }
]
```

Result: calls to `GET /v1/secrets/*path` can fail authorization because this endpoint requires `decrypt`.

Fixed policy:

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["decrypt"]
  }
]
```

Also verify path matching, for example `/v1/secrets/app/prod/*` if you want tighter scope.

## Common policy mistakes

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `403` on `GET /v1/secrets/*path` | Used `read` instead of `decrypt` | Grant `decrypt` for the secret path |
| `403` on transit rotate | Missing `rotate` capability | Add `rotate` on `/v1/transit/keys/*/rotate` |
| `403` on tokenization detokenize | Used `read` instead of `decrypt` | Grant `decrypt` on `/v1/tokenization/detokenize` |
| Service can access too much | Over-broad wildcard `*` path | Scope paths to service/environment prefixes |
| Writes fail on secrets endpoint | Used `write` instead of `encrypt` | Grant `encrypt` for `POST /v1/secrets/*path` |
| Tokenization lifecycle calls fail | Sent token in URL path policy scope only | Add explicit paths for `/v1/tokenization/detokenize`, `/v1/tokenization/validate`, and `/v1/tokenization/revoke` |
| Audit query denied | Missing `read` on `/v1/audit-logs` | Add explicit audit read policy |

## Best practices

1. Start with least privilege, then grant only what is required
2. Use separate clients per service/workload
3. Avoid wildcard `*` except emergency administration
4. Review policies on every deploy and rotation cycle

## See also

- [Authentication API](authentication.md)
- [Clients API](clients.md)
- [Capability matrix](capability-matrix.md)
- [Secrets API](secrets.md)
- [Transit API](transit.md)
