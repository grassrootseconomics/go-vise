// Example: Manual data storage using session id provided to engine.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	testdataloader "github.com/peteole/testdata-loader"

	fsdb "github.com/grassrootseconomics/go-vise/db/fs"
	"github.com/grassrootseconomics/go-vise/engine"
	"github.com/grassrootseconomics/go-vise/resource"
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

var (
	logg        = slogging.Get().With("component", "session-example")
	baseDir     = testdataloader.GetBasePath()
	scriptDir   = path.Join(baseDir, "examples", "session")
	emptyResult = resource.Result{}
)

func save(ctx context.Context, sym string, input []byte) (resource.Result, error) {
	sessionId := ctx.Value("SessionId").(string)
	sessionDir := path.Join(scriptDir, sessionId)
	err := os.MkdirAll(sessionDir, 0700)
	if err != nil {
		return emptyResult, err
	}
	fp := path.Join(sessionDir, "data.txt")
	if len(input) > 0 {
		logg.Debugf("write data %s session %s", input, sessionId)
		err = ioutil.WriteFile(fp, input, 0600)
		if err != nil {
			return emptyResult, err
		}
	}
	r, err := ioutil.ReadFile(fp)
	if err != nil {
		err = ioutil.WriteFile(fp, []byte("(not set)"), 0600)
		if err != nil {
			return emptyResult, err
		}
	}
	return resource.Result{
		Content: string(r),
	}, nil
}

func main() {
	var root string
	var size uint
	var sessionId string
	flag.UintVar(&size, "s", 0, "max size of output")
	flag.StringVar(&root, "root", "root", "entry point symbol")
	flag.StringVar(&sessionId, "session-id", "default", "session id")
	flag.Parse()
	fmt.Fprintf(os.Stderr, "starting session at symbol '%s' using resource dir: %s\n", root, scriptDir)

	ctx := context.Background()
	store := fsdb.NewFsDb()
	err := store.Connect(ctx, scriptDir)
	if err != nil {
		panic(err)
	}
	rs := resource.NewDbResource(store)
	rs.AddLocalFunc("do_save", save)
	cfg := engine.Config{
		Root:       "root",
		SessionId:  sessionId,
		OutputSize: uint32(size),
		StateDebug: true,
	}
	ctx = context.WithValue(ctx, "SessionId", sessionId)
	en := engine.NewEngine(cfg, rs)
	err = engine.Loop(ctx, en, os.Stdin, os.Stdout, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loop exited with error: %v\n", err)
		os.Exit(1)
	}
}
