package client

import (
	"encoding/json"
	"testing"
)

func TestPaginationParams_marshalsJSON(t *testing.T) {
	p := PaginationParams{Page: 0, PageSize: 50}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"page":0,"pageSize":50}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestPagination_unmarshalsJSON(t *testing.T) {
	in := `{"page":2,"pageSize":50,"total":312,"totalPages":7}`
	var p Pagination
	if err := json.Unmarshal([]byte(in), &p); err != nil {
		t.Fatal(err)
	}
	if p.Page != 2 || p.PageSize != 50 || p.Total != 312 || p.TotalPages != 7 {
		t.Errorf("got %+v", p)
	}
}
