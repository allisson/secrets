# 🧪 Code Examples

> Last updated: 2026-02-21

Complete code examples for integrating with Secrets APIs across multiple languages and releases.

## 📑 Quick Navigation

**By Language**:

- [Curl](curl.md) - Command-line examples
- [Python](python.md) - Python client examples
- [JavaScript](javascript.md) - Node.js client examples
- [Go](go.md) - Go client examples

**By Version**: See [Version Compatibility](#version-compatibility) below

---

## Version Compatibility

Use this section to quickly identify which example set matches your deployed release.

### Current release (`v0.10.0`)

- Primary examples:
  - [Curl examples](curl.md)
  - [Python examples](python.md)
  - [JavaScript examples](javascript.md)
  - [Go examples](go.md)
- Release context:
  - [v0.10.0 release notes](../releases/RELEASES.md#0100---2026-02-21)

### Previous release (`v0.7.0`)

- Backward context:
  - [v0.7.0 release notes](../releases/RELEASES.md#070---2026-02-20)

### Compatibility notes

- Example payloads and status codes follow current API docs (`/v1/*`)
- For endpoint-specific behavior changes, read release notes first
- For throttling behavior, validate `429` + `Retry-After` handling in your client runtime

---

## Getting Started

1. Choose your language from the list above
2. Check version compatibility if using an older release
3. Review authentication patterns (all examples use Bearer tokens)
4. Adapt examples to your use case

## See also

- [Authentication API](../api/auth/authentication.md)
- [API error decision matrix](../api/fundamentals.md#error-decision-matrix)
- [API rate limiting](../api/fundamentals.md#rate-limiting)
- [Release notes](../releases/RELEASES.md)
