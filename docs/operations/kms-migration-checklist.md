# ✅ KMS Migration Checklist

> Last updated: 2026-02-20

Use this checklist for migrating from legacy plaintext master keys to KMS mode.

## 1) Precheck

- [ ] Confirm target release is `v0.7.0` or newer
- [ ] Back up current environment configuration
- [ ] Confirm rollback owner and change window
- [ ] Confirm KMS provider credentials are available in runtime
- [ ] Confirm KMS encrypt/decrypt permissions are granted

## 2) Build KMS key chain

- [ ] Generate new KMS-encrypted key with `create-master-key --kms-provider ... --kms-key-uri ...`
- [ ] Re-encode existing legacy plaintext keys into KMS ciphertext
- [ ] Build `MASTER_KEYS` with only KMS ciphertext entries (no plaintext mix)
- [ ] Set `KMS_PROVIDER`, `KMS_KEY_URI`, and `ACTIVE_MASTER_KEY_ID`

Reference: [KMS setup guide](kms-setup.md#migration-from-legacy-mode)

## 3) Rollout

- [ ] Restart API instances (rolling)
- [ ] Verify startup logs show KMS mode and key decrypt lines
- [ ] Run baseline checks: `GET /health`, `GET /ready`
- [ ] Run key-dependent smoke checks: token issuance, secrets, transit

Reference: [Production rollout golden path](production-rollout.md)

## 4) Rotation and cleanup

- [ ] Rotate KEK after switching to KMS key chain
- [ ] Verify reads/decrypt for existing data still succeed
- [ ] Remove old key entries from `MASTER_KEYS` only after verification
- [ ] Restart API instances again after key-chain cleanup

Reference: [Key management operations](key-management.md)

## 5) Rollback readiness

- [ ] Keep previous image tag available
- [ ] Keep pre-change env snapshot available
- [ ] If rollback needed, revert app version first
- [ ] Re-validate health and smoke checks after rollback

Reference: [v0.7.0 upgrade guide](../releases/v0.7.0-upgrade.md#rollback-notes)

## See also

- [KMS setup guide](kms-setup.md)
- [Key management operations](key-management.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
