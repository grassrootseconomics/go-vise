package render

import (
	"context"
	"testing"
)

func TestMenuInit(t *testing.T) {
	m := NewMenu()
	err := m.Put("1", "foo")
	if err != nil {
		t.Fatal(err)
	}
	err = m.Put("2", "bar")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()
	r, err := m.Render(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	expect := `1:foo
2:bar`
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}

	r, err = m.Render(ctx, 1)
	if err == nil {
		t.Fatalf("expected render fail")
	}

}

func TestMenuBrowse(t *testing.T) {
	cfg := DefaultBrowseConfig()
	m := NewMenu().WithPageCount(3).WithBrowseConfig(cfg)
	err := m.Put("1", "foo")
	if err != nil {
		t.Fatal(err)
	}
	err = m.Put("2", "bar")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()
	r, err := m.Render(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	expect := `1:foo
2:bar
11:next`
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}

	r, err = m.Render(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	expect = `1:foo
2:bar
11:next
22:previous`
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}

	r, err = m.Render(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	expect = `1:foo
2:bar
22:previous`
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}

	_, err = m.Render(ctx, 3)
	if err == nil {
		t.Fatalf("expected render fail")
	}
}

func TestMenuSep(t *testing.T) {
	m := NewMenu().WithSeparator("//")
	err := m.Put("1", "foo")
	if err != nil {
		t.Fatal(err)
	}
	err = m.Put("2", "bar")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()
	r, err := m.Render(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	expect := `1//foo
2//bar`
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}
}
