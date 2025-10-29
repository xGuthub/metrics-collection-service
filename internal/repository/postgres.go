package repository

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStorage implements Storage backed by PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage connects to PostgreSQL using provided DSN and ensures schema exists.
func NewPostgresStorage(ctx context.Context, dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &PostgresStorage{db: db}
	if err := s.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (p *PostgresStorage) ensureSchema(ctx context.Context) error {
	// Two simple tables: gauges and counters
	if _, err := p.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gauges (
            name   TEXT PRIMARY KEY,
            value  DOUBLE PRECISION NOT NULL
        );`); err != nil {
		return err
	}
	if _, err := p.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS counters (
            name   TEXT PRIMARY KEY,
            value  BIGINT NOT NULL
        );`); err != nil {
		return err
	}
	return nil
}

func (p *PostgresStorage) Close() error { return p.db.Close() }

// Ping checks database connectivity.
func (p *PostgresStorage) Ping(ctx context.Context) error { return p.db.PingContext(ctx) }

func (p *PostgresStorage) UpdateGauge(name string, value float64) {
	// Use UPSERT to set the gauge value
	_, _ = p.db.Exec(`
        INSERT INTO gauges(name, value) VALUES($1, $2)
        ON CONFLICT(name) DO UPDATE SET value = EXCLUDED.value
    `, name, value)
}

func (p *PostgresStorage) UpdateCounter(name string, delta int64) {
	// Insert or add delta atomically via upsert
	_, _ = p.db.Exec(`
        INSERT INTO counters(name, value) VALUES($1, $2)
        ON CONFLICT(name) DO UPDATE SET value = counters.value + EXCLUDED.value
    `, name, delta)
}

func (p *PostgresStorage) GetGauge(name string) (float64, bool) {
	var v float64
	err := p.db.QueryRow(`SELECT value FROM gauges WHERE name=$1`, name).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false
		}
		return 0, false
	}
	return v, true
}

func (p *PostgresStorage) GetCounter(name string) (int64, bool) {
	var v int64
	err := p.db.QueryRow(`SELECT value FROM counters WHERE name=$1`, name).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false
		}
		return 0, false
	}
	return v, true
}

func (p *PostgresStorage) AllGauges() map[string]float64 {
	rows, err := p.db.Query(`SELECT name, value FROM gauges`)
	if err != nil {
		return map[string]float64{}
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err == nil {
			out[name] = val
		}
	}
	if err := rows.Err(); err != nil {
		return map[string]float64{}
	}
	return out
}

func (p *PostgresStorage) AllCounters() map[string]int64 {
	rows, err := p.db.Query(`SELECT name, value FROM counters`)
	if err != nil {
		return map[string]int64{}
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var name string
		var val int64
		if err := rows.Scan(&name, &val); err == nil {
			out[name] = val
		}
	}
	if err := rows.Err(); err != nil {
		return map[string]int64{}
	}
	return out
}
