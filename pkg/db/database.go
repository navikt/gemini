package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nais/gemini/pkg/metrics"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNotFound = fmt.Errorf("database row not found")

	_ Database = &postgresDB{}
)

type postgresDB struct {
	conn *pgxpool.Pool
}

func IsErrNotFound(err error) bool {
	return err == ErrNotFound
}

// Returns true if the error message is a foreign key constraint violation
func IsErrForeignKeyViolation(err error) bool {
	return strings.Contains(err.Error(), "SQLSTATE 23503")
}

func New(ctx context.Context, dsn string) (Database, error) {
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &postgresDB{
		conn: conn,
	}, nil
}

func (db *postgresDB) timedQuery(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	now := time.Now()
	rows, err := db.conn.Query(ctx, sql, args...)
	metrics.DatabaseQuery(now, err)
	return rows, err
}

func (db *postgresDB) Migrate(ctx context.Context) error {
	var version int

	query := `SELECT MAX(version) FROM migrations`
	row := db.conn.QueryRow(ctx, query)
	err := row.Scan(&version)

	if err != nil {
		// error might be due to no schema.
		// no way to detect this, so log error and continue with migrations.
		log.Warnf("unable to get current migration version: %s", err)
	}

	for version < len(migrations) {
		log.Infof("migrating database schema to version %d", version+1)

		_, err = db.conn.Exec(ctx, migrations[version])
		if err != nil {
			return fmt.Errorf("migrating to version %d: %s", version+1, err)
		}

		version++
	}

	return nil
}
