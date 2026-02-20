# ⚠️ Known Limitations

> Last updated: 2026-02-20

This page documents practical limitations and tradeoffs operators should account for.

## Rate limiting

- Token endpoint rate limiting is per-IP; shared NAT/proxy egress can impact legitimate callers
- Header trust/proxy misconfiguration can skew caller IP behavior
- Application-level throttling complements but does not replace edge/WAF controls

## Proxy and source-IP trust

- If forwarded headers are over-trusted, source IP spoofing risk increases
- If trusted proxy chain is incomplete, all traffic may appear from one source

## KMS startup model

- KMS decryption happens at startup key-load time, not per-request
- Runtime KMS outages may not impact steady-state traffic immediately, but restart/redeploy can fail if KMS is unavailable

## Operational cadence

- Key rotation requires API restart/rolling restart to load new key material
- Cleanup routines (`clean-audit-logs`, `clean-expired-tokens`) are operator-driven

## Documentation scope note

- `docs/openapi.yaml` is a baseline subset, not exhaustive contract coverage for every workflow detail

## See also

- [Trusted proxy reference](trusted-proxy-reference.md)
- [Security hardening guide](security-hardening.md)
- [KMS setup guide](kms-setup.md)
- [Production deployment guide](production.md)
