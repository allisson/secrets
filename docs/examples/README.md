# 🧪 Code Examples

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

All examples in this directory target the latest stable release .

For full version-specific behavior changes and compatibility history, please refer to the consolidated release notes:

- 📦 [Full Release History](../../CHANGELOG.md)

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

- [Authentication API](../auth/authentication.md)
- [API error decision matrix](../concepts/api-fundamentals.md#error-decision-matrix)
- [API rate limiting](../concepts/api-fundamentals.md#rate-limiting)
- [Release notes](../../CHANGELOG.md)
