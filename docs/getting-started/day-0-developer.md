# 💻 Day 0 Developer Walkthrough

> Last updated: 2026-02-20

Use this path for first-time contributors integrating with Secrets APIs.

## Step 1: Run locally

- Follow: [Run locally](local-development.md)
- Build and start API, then verify health

## Step 2: Understand auth + policy behavior

- Read: [Authentication API](../api/authentication.md)
- Read: [Policies cookbook](../api/policies.md)
- Read: [Capability matrix](../api/capability-matrix.md)

## Step 3: Validate error and retry behavior

- Read: [API error decision matrix](../api/error-decision-matrix.md)
- Read: [API rate limiting](../api/rate-limiting.md)

## Step 4: Use examples by runtime

- Start with: [Versioned examples by release](../examples/versioned-by-release.md)
- Then use: [Curl](../examples/curl.md), [Python](../examples/python.md), [JavaScript](../examples/javascript.md), [Go](../examples/go.md)

## Step 5: Follow docs contribution quality bar

- Read: [Documentation contributing guide](../contributing.md)
- Use: [Docs release checklist](../development/docs-release-checklist.md)

## Expected outcomes

- You can obtain tokens and call protected endpoints reliably
- You can distinguish authn/authz/throttling failures in client integrations
- You can submit feature PRs with aligned API + docs changes

## See also

- [Testing guide](../development/testing.md)
- [Docs architecture map](../development/docs-architecture-map.md)
