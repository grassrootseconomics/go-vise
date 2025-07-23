package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grassrootseconomics/go-vise/db"
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

type (
	PgInterface interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
		Query(context.Context, string, ...any) (pgx.Rows, error)
		QueryRow(context.Context, string, ...any) pgx.Row
		Ping(context.Context) error
		Close()
	}

	queries struct {
		migrate string
		get     string
		put     string
	}

	// pgDb is a Postgres backend implementation of the Db interface.
	pgDb struct {
		*db.DbBase
		conn    PgInterface
		schema  string
		prefix  uint8
		prepd   bool
		it      pgx.Rows
		itBase  []byte
		logg    slogging.Logger
		queries *queries
		once    sync.Once
	}
)

var (
	ErrSchemaChange = errors.New("schema change after connection established")
	ErrNoConnection = errors.New("no database connection established")
	ErrUnsafePut    = errors.New("unsafe put operation with safety enabled")
)

// NewpgDb creates a new Postgres backed Db implementation.
func NewPgDb() *pgDb {
	db := &pgDb{
		DbBase: db.NewDbBase(),
		logg:   slogging.Get().With("component", "postgres"),
	}
	db.WithSchema("public")
	return db
}

// Base implements Db
func (pdb *pgDb) Base() *db.DbBase {
	return pdb.DbBase
}

// WithSchema sets the Postgres schema to use for the storage table.
func (pdb *pgDb) WithSchema(schema string) *pgDb {
	if pdb.prepd {
		pdb.logg.Errorf("cannot change schema after connection established due to prepared statement caching")
		// We starught up panic here to avoid any further issues.
		panic(ErrSchemaChange)
	} else {
		pdb.schema = schema
		pdb.updateQueries()
	}
	return pdb
}

func (pdb *pgDb) updateQueries() {
	pdb.queries = &queries{
		migrate: fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.kv_vise (id SERIAL NOT NULL, key BYTEA NOT NULL UNIQUE, value BYTEA NOT NULL, updated TIMESTAMP NOT NULL);", pdb.schema),
		get:     fmt.Sprintf("SELECT value FROM %s.kv_vise WHERE key = $1", pdb.schema),
		put:     fmt.Sprintf("INSERT INTO %s.kv_vise (key, value, updated) VALUES ($1, $2, 'now') ON CONFLICT(key) DO UPDATE SET value = $2, updated = 'now';", pdb.schema),
	}
}

func (pdb *pgDb) WithConnection(pi PgInterface) *pgDb {
	pdb.conn = pi
	return pdb
}

// Connect implements Db.
func (pdb *pgDb) Connect(ctx context.Context, connStr string) error {
	if pdb.conn != nil {
		pdb.logg.WarnCtxf(ctx, "Pg already connected")
		return nil
	}
	conn, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return err
	}

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("connection to postgres could not be established: %w", err)
	}

	pdb.conn = conn
	pdb.DbBase.Connect(ctx, connStr)
	return pdb.ensureTable(ctx)
}

func (pdb *pgDb) Start(ctx context.Context) error {
	pdb.logg.Warnf("Start() deprecated: use executeTransaction instead")
	return nil
}

func (pdb *pgDb) Stop(ctx context.Context) error {
	pdb.logg.Warnf("Stop() deprecated: use executeTransaction instead")
	return nil
}

func (pdb *pgDb) Abort(ctx context.Context) {
	pdb.logg.Warnf("Abort() deprecated: use executeTransaction instead")
}

// Put implements Db.
func (pdb *pgDb) Put(ctx context.Context, key []byte, val []byte) error {
	if !pdb.CheckPut() {
		return ErrUnsafePut
	}

	lk, err := pdb.ToKey(ctx, key)
	if err != nil {
		return err
	}

	pdb.logg.TraceCtxf(ctx, "put", "key", key, "val", val)
	actualKey := lk.Default
	if lk.Translation != nil {
		actualKey = lk.Translation
	}

	_, err = pdb.conn.Exec(ctx, pdb.queries.put, actualKey, val)
	return err
}

// Get implements Db.
func (pdb *pgDb) Get(ctx context.Context, key []byte) ([]byte, error) {
	var (
		rr         []byte
		queryParam []byte
	)

	lk, err := pdb.ToKey(ctx, key)
	if err != nil {
		return nil, err
	}

	pdb.logg.TraceCtxf(ctx, "get", "key", key)
	if lk.Translation != nil {
		queryParam = lk.Translation
	} else {
		queryParam = lk.Default
	}

	err = pdb.conn.QueryRow(ctx, pdb.queries.get, queryParam).Scan(&rr)
	if err != nil {
		return nil, err
	}

	return rr, nil
}

// Close implements Db.
func (pdb *pgDb) Close(ctx context.Context) error {
	if pdb.conn == nil {
		return ErrNoConnection
	}
	pdb.conn.Close()
	return nil
}

// set up table
func (pdb *pgDb) ensureTable(ctx context.Context) error {
	var err error
	pdb.once.Do(func() {
		if _, execErr := pdb.conn.Exec(ctx, pdb.queries.migrate); execErr != nil {
			err = fmt.Errorf("failed to ensure table exists: %w", execErr)
			return
		}
		pdb.prepd = true
	})

	if err != nil {
		return err
	}

	if !pdb.prepd {
		pdb.logg.WarnCtxf(ctx, "ensureTable called but table not prepared")
	}

	return nil
}
