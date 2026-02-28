# 🎨 Documentation Style Guide

> Last updated: 2026-02-28

This guide defines the writing style and technical conventions for Secrets documentation.

## Writing Style

- Use short, direct sentences.
- Use active voice and practical wording.
- Keep headings in Title Case.
- Use emojis for scanability, but keep usage moderate.
- Keep list items concise and without trailing periods.
- Prefer short sections with clear headings over long uninterrupted blocks.
- Prefer plain bullet lists and tables over heavily emphasized text blocks.
- Keep cross-links clickable (Markdown links) rather than inline code path references.
- Keep operational steps copy/paste-ready and include expected status/result when useful.

## Technical Accuracy

- Match implemented API paths exactly (`/v1/*`).
- Use capabilities consistently: `read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`.
- Include `Last updated` metadata in new docs.
- For API docs, include `Applies to: API v1`.

## Security Messaging

- Use this exact warning where base64 appears:
  - `⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.`
- Avoid security claims not backed by implementation.
- Do not reintroduce removed features into docs.

## Examples

- Prefer copy/paste-ready examples.
- Include expected status/result where useful.
- Avoid placeholder values that look like real secrets.

### Copy-safe examples policy

- Use clearly synthetic values (`<client-id>`, `tok_sample`, `example.com`).
- Never include real keys, tokens, credentials, or production hostnames.
- For sensitive domains (payments/PII), prefer redacted fragments (e.g., `last_four`).

## See Also

- [Contributing Guide](../contributing.md)
- [Documentation Process](docs-process.md)
