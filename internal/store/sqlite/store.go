package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }
type transaction struct{ tx *sql.Tx }

func Open(path string) (*Store, error) {
	if path == "" {
		path = "caption-audit.db"
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	store := &Store{db: db}
	if err := store.configure(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) configure(ctx context.Context) error {
	for _, statement := range []string{"PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000"} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("配置 SQLite: %w", err)
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Transact(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&transaction{tx: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 SQLite 事务: %w", err)
	}
	return nil
}

func (s *Store) CheckIntegrity(ctx context.Context) error {
	var result string
	if err := s.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite 完整性检查失败: %s", result)
	}
	return s.Transact(ctx, func(raw application.Tx) error {
		tx := raw.(*transaction)
		rows, err := tx.tx.QueryContext(ctx, "SELECT DISTINCT job_id FROM audit_events ORDER BY job_id")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			if err := tx.verifyAudit(id); err != nil {
				return err
			}
		}
		return rows.Err()
	})
}

var _ application.Repository = (*Store)(nil)
