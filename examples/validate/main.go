// Example: Input checker.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"

	testdataloader "github.com/peteole/testdata-loader"

	fsdb "github.com/grassrootseconomics/go-vise/db/fs"
	"github.com/grassrootseconomics/go-vise/engine"
	"github.com/grassrootseconomics/go-vise/resource"
	"github.com/grassrootseconomics/go-vise/state"
)

var (
	baseDir     = testdataloader.GetBasePath()
	scriptDir   = path.Join(baseDir, "examples", "validate")
	emptyResult = resource.Result{}
)

const (
	USERFLAG_HAVESOMETHING = state.FLAG_USERSTART
)

type verifyResource struct {
	*resource.DbResource
	st *state.State
}

func (vr *verifyResource) verify(ctx context.Context, sym string, input []byte) (resource.Result, error) {
	var err error
	if string(input) == "something" {
		vr.st.SetFlag(USERFLAG_HAVESOMETHING)
	}
	return resource.Result{
		Content: "",
	}, err
}

func (vr *verifyResource) again(ctx context.Context, sym string, input []byte) (resource.Result, error) {
	vr.st.ResetFlag(USERFLAG_HAVESOMETHING)
	return resource.Result{}, nil
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
	ctx = context.WithValue(ctx, "SessionId", sessionId)
	st := state.NewState(1)
	store := fsdb.NewFsDb()
	err := store.Connect(ctx, scriptDir)
	if err != nil {
		panic(err)
	}
	rsf := resource.NewDbResource(store)
	rs := verifyResource{rsf, st}
	rs.AddLocalFunc("verifyinput", rs.verify)
	rs.AddLocalFunc("again", rs.again)

	cfg := engine.Config{
		Root:       "root",
		SessionId:  sessionId,
		OutputSize: uint32(size),
	}

	en := engine.NewEngine(cfg, rs)
	en = en.WithState(st)
	err = engine.Loop(ctx, en, os.Stdin, os.Stdout, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loop exited with error: %v\n", err)
		os.Exit(1)
	}
}
