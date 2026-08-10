package database

import (
	"context"
	"database/sql"
	"time"
)

// ListLatestCursor returns, for each logical key, its latest non-deleted row,
// ordered by key ascending with cursor-based pagination. The two-branch
// latest-version JOIN, row iteration, and empty-slice normalisation live here
// so feature repositories don't hand-copy them.
//
// selectCols must be the SELECT column list for the OUTER query, each column
// prefixed with the alias t (e.g. "t.id, t.name, ..."). table and keyCol are
// the physical table name and its logical key column. scan maps one row to T.
//
// A nil afterKey returns the first page; otherwise rows with key > afterKey.
// Returns a non-nil empty slice when nothing matches.
func ListLatestCursor[T any](
	ctx context.Context,
	q Querier,
	table, keyCol, selectCols string,
	afterKey *string,
	limit int,
	scan func(*sql.Rows) (T, error),
) ([]T, error) {
	var query string
	var args []any

	if afterKey == nil {
		//nolint:gosec // table/keyCol/selectCols are compile-time constants from callers, not user input.
		query = `
			SELECT ` + selectCols + `
			FROM ` + table + ` t
			INNER JOIN (
				SELECT ` + keyCol + `, MAX(version) as max_version
				FROM ` + table + `
				WHERE deleted_at IS NULL
				GROUP BY ` + keyCol + `
				ORDER BY ` + keyCol + ` ASC
				LIMIT $1
			) latest ON t.` + keyCol + ` = latest.` + keyCol + ` AND t.version = latest.max_version
			ORDER BY t.` + keyCol + ` ASC`
		args = []any{limit}
	} else {
		//nolint:gosec // same as above: constant fragments, not user input.
		query = `
			SELECT ` + selectCols + `
			FROM ` + table + ` t
			INNER JOIN (
				SELECT ` + keyCol + `, MAX(version) as max_version
				FROM ` + table + `
				WHERE deleted_at IS NULL AND ` + keyCol + ` > $1
				GROUP BY ` + keyCol + `
				ORDER BY ` + keyCol + ` ASC
				LIMIT $2
			) latest ON t.` + keyCol + ` = latest.` + keyCol + ` AND t.version = latest.max_version
			ORDER BY t.` + keyCol + ` ASC`
		args = []any{*afterKey, limit}
	}

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]T, 0, limit)
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// HardDeleteOlderThan permanently removes soft-deleted rows older than the
// cutoff. In dryRun mode it returns the count without deleting.
func HardDeleteOlderThan(
	ctx context.Context,
	q Querier,
	table string,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	if dryRun {
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE deleted_at IS NOT NULL AND deleted_at < $1`
		var count int64
		if err := q.QueryRowContext(ctx, query, olderThan).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}

	query := `DELETE FROM ` + table + ` WHERE deleted_at IS NOT NULL AND deleted_at < $1`
	result, err := q.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, nil
}
