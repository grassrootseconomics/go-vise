// Example: Use gdbm backend to retrieve resources.
package main

import (
	"context"
	"fmt"
	"os"
	"path"

	testdataloader "github.com/peteole/testdata-loader"

	"github.com/grassrootseconomics/go-vise/db"
	fsdb "github.com/grassrootseconomics/go-vise/db/fs"
	gdbmdb "github.com/grassrootseconomics/go-vise/db/gdbm"
	"github.com/grassrootseconomics/go-vise/engine"
	"github.com/grassrootseconomics/go-vise/resource"
)

var (
	baseDir   = testdataloader.GetBasePath()
	scriptDir = path.Join(baseDir, "examples", "gdbm")
	dbFile    = path.Join(scriptDir, "vise.gdbm")
)

func do(ctx context.Context, sym string, input []byte) (resource.Result, error) {
	return resource.Result{
		Content: "bye",
	}, nil
}

func main() {
	ctx := context.Background()
	root := "root"
	fmt.Fprintf(os.Stderr, "starting session at symbol '%s' using resource dir: %s\n", root, scriptDir)

	store := gdbmdb.NewGdbmDb()
	err := store.Connect(ctx, dbFile)
	if err != nil {
		panic(err)
	}

	tg := resource.NewDbResource(store)
	tg.Without(db.DATATYPE_MENU)
	rs := resource.NewMenuResource()
	rs = rs.WithTemplateGetter(tg.GetTemplate)
	rs = rs.WithCodeGetter(tg.GetCode)

	fsStore := fsdb.NewFsDb()
	fsStore.Connect(ctx, scriptDir)
	rsf := resource.NewDbResource(fsStore)
	rsf.WithOnly(db.DATATYPE_MENU)
	rsf.AddLocalFunc("do", do)
	rs.WithMenuGetter(rsf.GetMenu)
	rs.WithEntryFuncGetter(rsf.FuncFor)

	cfg := engine.Config{
		Root:     "root",
		Language: "nor",
	}
	en := engine.NewEngine(cfg, rs)

	err = engine.Loop(ctx, en, os.Stdin, os.Stdout, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loop exited with error: %v\n", err)
		os.Exit(1)
	}
}
