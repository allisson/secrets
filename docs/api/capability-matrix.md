# 🗂️ Capability Matrix

> Last updated: 2026-02-19
> Applies to: API v1

This page is the canonical capability-to-endpoint reference used by API docs and policy templates.

## Capability Definitions

- `read`: list or inspect metadata/state without decrypting payload values
- `write`: create or update non-cryptographic resources and key definitions
- `delete`: delete resources or revoke token lifecycle entries
- `encrypt`: create encrypted outputs (secrets writes, transit encrypt, tokenization tokenize)
- `decrypt`: resolve encrypted/tokenized values back to plaintext
- `rotate`: create new key versions

## Endpoint Matrix

| Endpoint | Required capability |
| --- | --- |
| `POST /v1/clients` | `write` |
| `GET /v1/clients` | `read` |
| `GET /v1/clients/:id` | `read` |
| `PUT /v1/clients/:id` | `write` |
| `DELETE /v1/clients/:id` | `delete` |
| `GET /v1/audit-logs` | `read` |
| `POST /v1/secrets/*path` | `encrypt` |
| `GET /v1/secrets/*path` | `decrypt` |
| `DELETE /v1/secrets/*path` | `delete` |
| `POST /v1/transit/keys` | `write` |
| `POST /v1/transit/keys/:name/rotate` | `rotate` |
| `DELETE /v1/transit/keys/:id` | `delete` |
| `POST /v1/transit/keys/:name/encrypt` | `encrypt` |
| `POST /v1/transit/keys/:name/decrypt` | `decrypt` |
| `POST /v1/tokenization/keys` | `write` |
| `POST /v1/tokenization/keys/:name/rotate` | `rotate` |
| `DELETE /v1/tokenization/keys/:id` | `delete` |
| `POST /v1/tokenization/keys/:name/tokenize` | `encrypt` |
| `POST /v1/tokenization/detokenize` | `decrypt` |
| `POST /v1/tokenization/validate` | `read` |
| `POST /v1/tokenization/revoke` | `delete` |

## Policy Authoring Notes

Policy matcher quick reference:

| Pattern type | Example | Matching behavior |
| --- | --- | --- |
| Exact | `/v1/audit-logs` | Only that exact path |
| Full wildcard | `*` | Any request path |
| Trailing wildcard | `/v1/secrets/*` | Prefix + nested paths |
| Mid-path wildcard | `/v1/transit/keys/*/rotate` | `*` matches one segment |

For complete matcher semantics and unsupported forms, see [Policies cookbook](policies.md#path-matching-behavior).

- Use path scope as narrowly as possible (service + environment prefixes).
- Avoid wildcard `*` except temporary break-glass workflows.
- Keep encrypt and decrypt separated across clients when operationally possible.
- For tokenization lifecycle endpoints, token value is passed in JSON body; policy path is endpoint path.

## See also

- [Policies cookbook](policies.md)
- [Authentication API](authentication.md)
- [Clients API](clients.md)
- [Secrets API](secrets.md)
- [Transit API](transit.md)
- [Tokenization API](tokenization.md)
