package cursor

import (
	"context"
	"errors"
	"testing"
)

type item struct{ ID int }

func collect[K any](t *testing.T, p *Paginator[item, K], limit int) []int {
	t.Helper()
	var ids []int
	cursor := ""
	for i := 0; i < 1000; i++ {
		pg, err := p.Page(context.Background(), cursor, limit)
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range pg.Data {
			ids = append(ids, it.ID)
		}
		if !pg.HasNext {
			return ids
		}
		cursor = pg.NextCursor
	}
	t.Fatal("loop không kết thúc")
	return nil
}

func data(n int) []item {
	out := make([]item, n)
	for i := range out {
		out[i] = item{ID: i + 1}
	}
	return out
}

func check(t *testing.T, ids []int, n int) {
	t.Helper()
	if len(ids) != n {
		t.Fatalf("want %d items, got %d", n, len(ids))
	}
	for i, id := range ids {
		if id != i+1 {
			t.Fatalf("sai thứ tự tại %d: %d", i, id)
		}
	}
}

func TestSlice(t *testing.T) {
	for _, n := range []int{0, 1, 10, 25} { // gồm cả biên: rỗng, đúng bội limit
		src := NewSliceSource(data(n), func(i item) int { return i.ID })
		check(t, collect(t, New[item, int](src, NewCodec([]byte("k"))), 5), n)
	}
}

func TestOffset(t *testing.T) {
	all := data(23)
	src := NewOffsetSource(func(_ context.Context, off, lim int) ([]item, error) {
		if off > len(all) {
			off = len(all)
		}
		end := off + lim
		if end > len(all) {
			end = len(all)
		}
		return all[off:end], nil
	})
	check(t, collect(t, New[item, int](src, nil), 10), 23)
}

func TestToken(t *testing.T) {
	all := data(12)
	src := NewTokenSource(func(_ context.Context, tok string, lim int) ([]item, string, error) {
		start := 0
		if tok != "" {
			for i := range all {
				if "t"+string(rune('a'+i)) == tok {
					start = i
				}
			}
		}
		end := start + lim
		if end >= len(all) {
			return all[start:], "", nil
		}
		return all[start:end], "t" + string(rune('a'+end)), nil
	})
	check(t, collect(t, New[item, string](src, nil), 5), 12)
}

func TestTamperedCursor(t *testing.T) {
	src := NewSliceSource(data(30), func(i item) int { return i.ID })
	p := New[item, int](src, NewCodec([]byte("k")))
	pg, _ := p.Page(context.Background(), "", 5)
	_, err := p.Page(context.Background(), pg.NextCursor+"x", 5)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("want ErrInvalidCursor, got %v", err)
	}
}
