# 🧭 Day 0 Operator Walkthrough

> Last updated: 2026-02-20

Use this linear path for first-time operations onboarding.

## Step 1: Bring up a local baseline

- Follow: [Run with Docker](docker.md)
- Verify: `GET /health`, `GET /ready`

## Step 2: Validate core flows

- Run: [Smoke test script](smoke-test.md)
- Confirm token issuance, secrets, and transit checks pass

## Step 3: Learn rollout and rollback flow

- Read: [Production rollout golden path](../operations/production-rollout.md)
- Focus: verification gates and rollback triggers

## Step 4: Learn incident response path

- Use: [Incident decision tree](../operations/incident-decision-tree.md)
- Drill: [First 15 Minutes Playbook](../operations/first-15-minutes.md)

## Step 5: Harden production posture

- Read: [Production deployment guide](../operations/production.md)
- Read: [Security hardening guide](../operations/security-hardening.md)
- Check: [Known limitations](../operations/known-limitations.md)

## Expected outcomes

- You can validate service health and auth quickly
- You can identify `401/403/429/5xx` primary runbook path
- You can execute a basic rollback trigger decision under pressure

## See also

- [Operator quick card](../operations/operator-quick-card.md)
- [Operator runbook index](../operations/runbook-index.md)
