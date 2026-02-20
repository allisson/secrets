# ⬆️ Upgrade Guide: vA.B.C -> vX.Y.Z

> Release date: YYYY-MM-DD

Use this guide to safely upgrade from `vA.B.C` to `vX.Y.Z`.

## Scope

- Release type: patch/minor/major
- API compatibility: compatible/incompatible notes
- Database migration: required/optional/none

## What Changed

- Change 1
- Change 2
- Change 3

## Env Diff (copy/paste)

```diff
+ NEW_VAR=value
- OLD_VAR=old-value
```

## Recommended Upgrade Steps

1. Update image/binary to `vX.Y.Z`
2. Apply env/config changes
3. Restart/roll instances
4. Run health checks
5. Run functional smoke checks

## Quick Verification Commands

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready
```

## Rollback Notes

- Revert to previous stable version first
- Keep non-destructive config rollback path documented
- Re-run validation after rollback

### Rollback matrix

| Upgrade path | First rollback action | Config rollback | Validation |
| --- | --- | --- | --- |
| `vA.B.C -> vX.Y.Z` | Roll app image/binary back | Revert/ignore release-specific config additions | Health + smoke checks |

## See also

- [Release notes template](_template.md)
- [Release compatibility matrix](compatibility-matrix.md)
