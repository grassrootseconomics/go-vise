package postgres

import (
	"context"
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grassrootseconomics/go-vise/db"
	"github.com/grassrootseconomics/go-vise/logging"
)

type (
	PgInterface interface {
		Begin(context.Context) (pgx.Tx, error)
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
		conn   PgInterface
		schema string
		prefix uint8
		prepd  bool
		it     pgx.Rows
		itBase []byte
		// tx      pgx.Tx
		// multi   bool
		logg    logging.Logger
		queries *queries
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
	}
	db.WithSchema("public")
	db.SetLogger(nil)
	return db
}

// Base implements Db
func (pdb *pgDb) Base() *db.DbBase {
	return pdb.DbBase
}

// WithSchema sets the Postgres schema to use for the storage table.
func (pdb *pgDb) WithSchema(schema string) *pgDb {
	if pdb.prepd {
		pdb.logg.Warnf("cannot change schema after connection established due to prepared statement caching")
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

func (pdb *pgDb) executeTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := pdb.conn.Begin(ctx)
	pdb.logg.TraceCtxf(ctx, "begin tx", "err", err)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			pdb.logg.TraceCtxf(ctx, "rollback tx", "err", err)
			tx.Rollback(ctx)
		} else {
			pdb.logg.TraceCtxf(ctx, "commit tx", "err", err)
			tx.Commit(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return nil
}

func (pdb *pgDb) Start(ctx context.Context) error {
	pdb.logg.Warnf("Start() deprecated: use executeTransaction instead")
	return nil

	// if pdb.tx != nil {
	// 	return db.ErrTxExist
	// }
	// err := pdb.start(ctx)
	// if err != nil {
	// 	return err
	// }
	// pdb.multi = true
	// return nil
}

func (pdb *pgDb) start(ctx context.Context) error {
	pdb.logg.Warnf("start() deprecated: use executeTransaction instead")
	return nil

	// if pdb.tx != nil {
	// 	return nil
	// }
	// tx, err := pdb.conn.BeginTx(ctx, defaultTxOptions)
	// pdb.logg.TraceCtxf(ctx, "begin single tx", "err", err)
	// if err != nil {
	// 	return err
	// }
	// pdb.tx = tx
	// return nil
}

func (pdb *pgDb) Stop(ctx context.Context) error {
	pdb.logg.Warnf("Stop() deprecated: use executeTransaction instead")
	return nil

	// if !pdb.multi {
	// 	return db.ErrSingleTx
	// }
	// return pdb.stop(ctx)
}

func (pdb *pgDb) stopSingle(ctx context.Context) error {
	pdb.logg.Warnf("stopSingle() deprecated: use executeTransaction instead")
	return nil
	// if pdb.multi {
	// 	return nil
	// }
	// err := pdb.tx.Commit(ctx)
	// pdb.logg.TraceCtxf(ctx, "stop single tx", "err", err)
	// pdb.tx = nil
	// return err
}

func (pdb *pgDb) stop(ctx context.Context) error {
	pdb.logg.Warnf("stop() deprecated: use executeTransaction instead")
	return nil
	// if pdb.tx == nil {
	// 	return db.ErrNoTx
	// }
	// err := pdb.tx.Commit(ctx)
	// pdb.logg.TraceCtxf(ctx, "stop multi tx", "err", err)
	// pdb.tx = nil
	// return err
}

func (pdb *pgDb) Abort(ctx context.Context) {
	pdb.logg.Warnf("Abort() deprecated: use executeTransaction instead")
	return
	// pdb.logg.InfoCtxf(ctx, "aborting tx", "tx", pdb.tx)
	// pdb.tx.Rollback(ctx)
	// pdb.tx = nil
}

// Put implements Db.
func (pdb *pgDb) Put(ctx context.Context, key []byte, val []byte) error {
	if !pdb.CheckPut() {
		return errors.New("unsafe put and safety set")
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

	return pdb.executeTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, pdb.queries.put, actualKey, val)
		return err
	})
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

	if err := pdb.executeTransaction(ctx, func(tx pgx.Tx) error {
		pdb.logg.TraceCtxf(ctx, "get", "key", key)

		if lk.Translation != nil {
			queryParam = lk.Translation
		} else {
			queryParam = lk.Default
		}

		return tx.QueryRow(ctx, pdb.queries.get, queryParam).Scan(&rr)
	}); err != nil {
		return nil, err
	}

	return rr, nil
}

// Close implements Db.
func (pdb *pgDb) Close(ctx context.Context) error {
	// err := pdb.Stop(ctx)
	// if err == db.ErrNoTx {
	// 	err = nil
	// }
	pdb.conn.Close()
	return nil
}

// set up table
// set up table
func (pdb *pgDb) ensureTable(ctx context.Context) error {
	if pdb.prepd {
		pdb.logg.WarnCtxf(ctx, "ensureTable called more than once")
		return nil
	}

	if err := pdb.executeTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, pdb.queries.migrate)

		return err
	}); err != nil {
		return err
	}

	return nil
}
