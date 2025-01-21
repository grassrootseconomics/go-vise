package state

import (
	"fmt"
	"strings"
	"testing"
)

func TestDebugFlagDenied(t *testing.T) {
	err := FlagDebugger.Register(7, "foo")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDebugFlagString(t *testing.T) {
	err := FlagDebugger.Register(8, "FOO")
	if err != nil {
		t.Fatal(err)
	}
	err = FlagDebugger.Register(9, "BAR")
	if err != nil {
		t.Fatal(err)
	}
	err = FlagDebugger.Register(11, "BAZ")
	if err != nil {
		t.Fatal(err)
	}
	flags := []byte{0x06, 0x19}
	r := FlagDebugger.AsString(flags, 5)
	expect := "INTERNAL_INMATCH(1),INTERNAL_WAIT(2),FOO(8),BAZ(11),?unreg?(12)"
	if r != expect {
		t.Fatalf("expected '%s', got '%s'", expect, r)
	}
}

func TestDebugState(t *testing.T) {
	err := FlagDebugger.Register(8, "FOO")
	if err != nil {
		t.Fatal(err)
	}
	st := NewState(1)
	st.UseDebug()
	st.SetFlag(FLAG_DIRTY)
	st.SetFlag(8)
	st.Down("root")

	r := fmt.Sprintf("%s", st)
	expect := "moves: 1 idx: 0 flags: INTERNAL_DIRTY(4),FOO(8) path: root lang: (default)"
	if strings.Contains(expect, r) {
		t.Fatalf("expected '%s', got '%s'", expect, r)
	}
}
