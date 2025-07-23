package postgres

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/grassrootseconomics/go-vise/db"
	"github.com/grassrootseconomics/go-vise/db/dbtest"
)

var (
	typMap = pgtype.NewMap()

	mockVfd = pgconn.FieldDescription{
		Name:        "value",
		DataTypeOID: pgtype.ByteaOID,
		Format:      typMap.FormatCodeForOID(pgtype.ByteaOID),
	}
)

func TestCasesPg(t *testing.T) {
	ctx := context.Background()

	t.Skip("implement expects in all cases")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")

	err = dbtest.RunTests(t, ctx, store)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPutGetPg(t *testing.T) {
	var dbi db.Db
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	dbi = store
	_ = dbi

	k := []byte("foo")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)
	v := []byte("bar")
	resInsert := pgxmock.NewResult("UPDATE", 1)
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}

	row := pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	b, err := store.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: implement as pgtype map instead, and btw also ask why getting base64 here
	if !bytes.Equal(b, v) {
		t.Fatalf("expected 'bar', got %x", b)
	}

	v = []byte("plugh")
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}

	row = pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	b, err = store.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, v) {
		t.Fatalf("expected 'plugh', got %x", b)
	}

}

func TestPostgresTxAbort(t *testing.T) {
	var dbi db.Db
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	dbi = store
	_ = dbi

	resInsert := pgxmock.NewResult("UPDATE", 1)
	k := []byte("foo")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)
	v := []byte("bar")
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}
	store.Abort(ctx)
}

func TestPostgresTxCommitOnClose(t *testing.T) {
	var dbi db.Db
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	dbi = store
	_ = dbi

	resInsert := pgxmock.NewResult("UPDATE", 1)
	k := []byte("foo")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)
	v := []byte("bar")

	ktwo := []byte("blinky")
	kstwo := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	kstwo = append(kstwo, []byte(".")...)
	kstwo = append(kstwo, ktwo...)
	vtwo := []byte("clyde")

	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(kstwo, vtwo).WillReturnResult(resInsert)

	err = store.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(ctx, ktwo, vtwo)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}

	row := pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	row = pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(vtwo)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(kstwo).WillReturnRows(row)

	store = NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	v, err = store.Get(ctx, k)
	if err != nil {
		if !db.IsNotFound(err) {
			t.Fatalf("get key one: %x", k)
		}
	}
	v, err = store.Get(ctx, ktwo)
	if err != nil {
		if !db.IsNotFound(err) {
			t.Fatalf("get key two: %x", ktwo)
		}
	}
}

func TestPostgresTxStartStop(t *testing.T) {
	var dbi db.Db
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	dbi = store
	_ = dbi

	resInsert := pgxmock.NewResult("UPDATE", 1)
	k := []byte("inky")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)
	v := []byte("pinky")

	ktwo := []byte("blinky")
	kstwo := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	kstwo = append(kstwo, []byte(".")...)
	kstwo = append(kstwo, ktwo...)
	vtwo := []byte("clyde")
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(kstwo, vtwo).WillReturnResult(resInsert)

	row := pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	row = pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(vtwo)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(kstwo).WillReturnRows(row)

	err = store.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(ctx, ktwo, vtwo)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Stop(ctx)
	if err != nil {
		t.Fatal(err)
	}

	v, err = store.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}
	v, err = store.Get(ctx, ktwo)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetNonExistentKey(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	k := []byte("nonexistent")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)

	// Mock expects a query that returns no rows (pgx.ErrNoRows)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnError(pgx.ErrNoRows)

	_, err = store.Get(ctx, k)
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}

	if !db.IsNotFound(err) {
		t.Fatalf("expected IsNotFound error, got: %v", err)
	}
}

func TestPutUnsafeOperation(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_MENU)
	store.SetSession(ses)
	store.SetLock(db.DATATYPE_MENU, true)
	ctx := context.Background()

	k := []byte("foo")
	v := []byte("bar")

	// No mock expectations since the operation should fail before hitting the database
	err = store.Put(ctx, k, v)
	if err == nil {
		t.Fatal("expected ErrUnsafePut")
	}

	if err != ErrUnsafePut {
		t.Fatalf("expected ErrUnsafePut, got: %v", err)
	}
}

func TestPutGetDifferentDataTypes(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	ctx := context.Background()

	k := []byte("foo")
	v := []byte("bar")
	resInsert := pgxmock.NewResult("UPDATE", 1)

	store.SetPrefix(db.DATATYPE_STATE)
	store.SetSession(ses)

	ks := append([]byte{db.DATATYPE_STATE}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)

	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}

	row := pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	b, err := store.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, v) {
		t.Fatalf("expected %x, got %x", v, b)
	}
}

func TestPutGetEmptyValues(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	// Test with empty value
	k := []byte("foo")
	v := []byte{}
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)

	resInsert := pgxmock.NewResult("UPDATE", 1)
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnResult(resInsert)
	err = store.Put(ctx, k, v)
	if err != nil {
		t.Fatal(err)
	}

	row := pgxmock.NewRowsWithColumnDefinition(mockVfd)
	row = row.AddRow(v)
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnRows(row)
	b, err := store.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, v) {
		t.Fatalf("expected empty value, got %x", b)
	}
}

func TestPutDatabaseError(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	k := []byte("foo")
	v := []byte("bar")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)

	// Mock a database error
	dbErr := errors.New("database connection failed")
	mock.ExpectExec("INSERT INTO vvise.kv_vise").WithArgs(ks, v).WillReturnError(dbErr)

	err = store.Put(ctx, k, v)
	if err == nil {
		t.Fatal("expected database error")
	}

	if err != dbErr {
		t.Fatalf("expected database error, got: %v", err)
	}
}

func TestGetDatabaseError(t *testing.T) {
	ses := "xyzzy"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	store := NewPgDb().WithConnection(mock).WithSchema("vvise")
	store.SetPrefix(db.DATATYPE_USERDATA)
	store.SetSession(ses)
	ctx := context.Background()

	k := []byte("foo")
	ks := append([]byte{db.DATATYPE_USERDATA}, []byte(ses)...)
	ks = append(ks, []byte(".")...)
	ks = append(ks, k...)

	// Mock a database error (not ErrNoRows)
	dbErr := errors.New("database connection failed")
	mock.ExpectQuery("SELECT value FROM vvise.kv_vise").WithArgs(ks).WillReturnError(dbErr)

	_, err = store.Get(ctx, k)
	if err == nil {
		t.Fatal("expected database error")
	}

	if err != dbErr {
		t.Fatalf("expected database error, got: %v", err)
	}
}
