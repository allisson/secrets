# 📘 Authorization Policy Cookbook

> Last updated: 2026-02-19
> Applies to: API v1

Ready-to-use policy templates for common service roles.

## 📑 Table of Contents

- [Policy structure](#policy-structure)
- [Path matching behavior](#path-matching-behavior)
- [Route shape vs policy shape](#route-shape-vs-policy-shape)
- [Policy review checklist before deploy](#policy-review-checklist-before-deploy)
- [Persona policy templates](#persona-policy-templates)
- [1) Read-only service](#1-read-only-service)
- [2) CI writer](#2-ci-writer)
- [3) Transit encrypt-only service](#3-transit-encrypt-only-service)
- [4) Transit decrypt-only service](#4-transit-decrypt-only-service)
- [5) Audit log reader](#5-audit-log-reader)
- [6) Break-glass admin (emergency)](#6-break-glass-admin-emergency)
- [7) Key operator](#7-key-operator)
- [8) Tokenization operator](#8-tokenization-operator)
- [Copy-safe split-role snippets](#copy-safe-split-role-snippets)
- [Pre-deploy policy automation](#pre-deploy-policy-automation)
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
    "path": "/v1/audit-logs",
    "capabilities": ["read"]
  }
]
```

Capabilities: `read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`.

## Path matching behavior

Policies are evaluated with case-sensitive matching rules:

- Exact path: no wildcard means full exact match (`/v1/audit-logs` matches only `/v1/audit-logs`)
- Full wildcard: `*` matches any request path
- Trailing wildcard: `prefix/*` matches paths starting with `prefix/` (greedy for deeper paths)
- Mid-path wildcard: `*` inside a path matches exactly one segment

Examples:

- `/v1/secrets/*` matches `/v1/secrets/app`, `/v1/secrets/app/db`, and `/v1/secrets/app/db/password`
- `/v1/transit/keys/*/rotate` matches `/v1/transit/keys/payment/rotate`
- `/v1/transit/keys/*/rotate` does not match `/v1/transit/keys/rotate` (missing segment)
- `/v1/transit/keys/*/rotate` does not match `/v1/transit/keys/payment/extra/rotate` (extra segment)
- `/v1/*/keys/*/rotate` matches `/v1/transit/keys/payment/rotate`

Unsupported patterns (not shell globs):

- Partial-segment wildcard like `/v1/transit/keys/prod-*`
- Suffix/prefix wildcard inside one segment like `*prod` or `prod*`
- Mixed-segment glob forms like `/v1/**/rotate`

## Route shape vs policy shape

- Route shape is validated by the HTTP router first (`404` on non-existent endpoint patterns).
- Policy shape is evaluated after route resolution (`403` when capability/path policy denies access).
- Example: `POST /v1/transit/keys/payment/extra/rotate` is a route-shape mismatch (`404`) before policy checks.
- Example: `POST /v1/transit/keys/payment/rotate` can still return `403` if caller lacks `rotate` on
  `/v1/transit/keys/*/rotate`.

Use [Policy smoke tests](../operations/policy-smoke-tests.md) to validate both route shape and policy behavior.

## Policy review checklist before deploy

1. Confirm endpoint capability intent for each path (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`).
2. Confirm wildcard type is intentional (exact, full `*`, trailing `/*`, or mid-path segment `*`).
3. Reject unsupported patterns (`prod-*`, `*prod`, `prod*`, `**`) before policy rollout.
4. Run route-shape and allow/deny smoke checks from [Policy smoke tests](../operations/policy-smoke-tests.md).
5. Review denied audit events after rollout and verify mismatches are expected.

Endpoint capability intent (quick map, condensed from [Capability matrix](capability-matrix.md)):

| Endpoint family | Typical capability |
| --- | --- |
| `GET /v1/clients`, `GET /v1/audit-logs`, `POST /v1/tokenization/validate` | `read` |
| `POST /v1/clients`, `PUT /v1/clients/:id`, `POST /v1/transit/keys`, `POST /v1/tokenization/keys` | `write` |
| `DELETE /v1/clients/:id`, `DELETE /v1/transit/keys/:id`, `DELETE /v1/tokenization/keys/:id`, `POST /v1/tokenization/revoke` | `delete` |
| `POST /v1/secrets/*path`, `POST /v1/transit/keys/:name/encrypt`, `POST /v1/tokenization/keys/:name/tokenize` | `encrypt` |
| `GET /v1/secrets/*path`, `POST /v1/transit/keys/:name/decrypt`, `POST /v1/tokenization/detokenize` | `decrypt` |
| `POST /v1/transit/keys/:name/rotate`, `POST /v1/tokenization/keys/:name/rotate` | `rotate` |

## Persona policy templates

Use these as starter profiles for common operational personas.

| Persona | Primary scope | Starter policy section |
| --- | --- | --- |
| Secrets reader | Read existing secrets only | [1) Read-only service](#1-read-only-service) |
| Secrets writer | CI/CD publish path | [2) CI writer](#2-ci-writer) |
| Transit encrypt worker | Encrypt-only workloads | [3) Transit encrypt-only service](#3-transit-encrypt-only-service) |
| Transit decrypt worker | Controlled decrypt runtime | [4) Transit decrypt-only service](#4-transit-decrypt-only-service) |
| Audit/compliance reader | Audit log retrieval | [5) Audit log reader](#5-audit-log-reader) |
| Key operator | Transit/tokenization key lifecycle | [7) Key operator](#7-key-operator) + [8) Tokenization operator](#8-tokenization-operator) |
| Break-glass admin | Emergency broad access | [6) Break-glass admin (emergency)](#6-break-glass-admin-emergency) |

Persona composition tips:

- Prefer one persona per client credential
- Keep encrypt/decrypt split across separate clients where possible
- Reserve wildcard `*` for short-lived emergency workflows only

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

Risk note: scope key names by environment with supported matchers. Use explicit key-name paths or
segment wildcards (for example `/v1/transit/keys/*/rotate`), not partial-segment wildcards like
`prod-*`.

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

## Copy-safe split-role snippets

Transit rotate-only operator:

```json
[
  {
    "path": "/v1/transit/keys/*/rotate",
    "capabilities": ["rotate"]
  }
]
```

Secrets read-only workload (`decrypt` only):

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["decrypt"]
  }
]
```

Secrets write-only workload (`encrypt` only):

```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["encrypt"]
  }
]
```

## Pre-deploy policy automation

Use this pre-deploy gate in CI to reject obvious policy mistakes before rollout.

```bash
#!/usr/bin/env bash
set -euo pipefail

POLICY_JSON_PATH="${1:-policy.json}"

# 1) Basic JSON validation
jq empty "$POLICY_JSON_PATH"

# 2) Reject unsupported wildcard forms in path segments
if jq -e '.[] | select(.path | test("\*\*|\w-\*|\*\w"))' "$POLICY_JSON_PATH" >/dev/null; then
  echo "unsupported wildcard pattern found in policy path"
  exit 1
fi

# 3) Ensure capabilities are from allowed set
ALLOWED='["read","write","delete","encrypt","decrypt","rotate"]'
if jq -e --argjson allowed "$ALLOWED" '.[] | .capabilities[] | select(($allowed | index(.)) == null)' "$POLICY_JSON_PATH" >/dev/null; then
  echo "unsupported capability found"
  exit 1
fi

echo "policy static checks: PASS"
```

For runtime allow/deny assertions, run [Policy smoke tests](../operations/policy-smoke-tests.md).

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
- [API error decision matrix](error-decision-matrix.md)
- [Clients API](clients.md)
- [Capability matrix](capability-matrix.md)
- [Secrets API](secrets.md)
- [Transit API](transit.md)
