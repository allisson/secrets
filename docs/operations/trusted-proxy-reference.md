# 🌐 Trusted Proxy Reference

> Last updated: 2026-02-20

Use this guide to validate source-IP forwarding for security controls that depend on caller IP
(for example token endpoint per-IP rate limiting on `POST /v1/token`).

## Why this matters

- If proxy trust is too broad, attackers may spoof `X-Forwarded-For`
- If proxy trust is too narrow/incorrect, many clients can collapse into one apparent IP
- Both cases can invalidate per-IP rate-limiting behavior

## Validation checklist

1. Only trusted edge proxies can set forwarded client-IP headers
2. Untrusted internet clients cannot inject arbitrary `X-Forwarded-For`
3. App-observed `client_ip` matches edge-proxy access logs for sampled requests
4. Multi-hop proxy behavior (if any) is documented and tested

## Nginx baseline forwarding

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Hardening notes:

- Do not accept forwarded headers directly from public clients
- Ensure only your reverse-proxy tier can reach application port `8080`

## AWS ALB / ELB notes

- ALB injects `X-Forwarded-For`; keep app reachable only from ALB/security group path
- Validate that downstream proxies preserve rather than overwrite trusted header chain
- Sample and compare ALB access logs with app `client_ip` logs

## Cloudflare / CDN edge notes

- Prefer single trusted edge path to origin
- If using CDN-specific client IP headers, keep mapping and validation documented
- Reject direct origin traffic from non-edge sources where possible

## Diagnostic quick test

1. Send a test request through edge proxy
2. Capture edge log source IP
3. Capture app log `client_ip` and request ID
4. Confirm both values refer to the same caller context

## Common failure patterns

- **All token requests share one IP:** likely NAT/proxy collapse or missing forwarded IP propagation
- **Frequent token `429` after proxy changes:** trust chain or source-IP extraction behavior drifted
- **Suspiciously diverse token caller IPs from one source:** potential forwarded-header spoofing

## See also

- [Security hardening guide](security-hardening.md)
- [Production deployment guide](production.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
