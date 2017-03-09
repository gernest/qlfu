package main

import (
	"strings"
	"testing"
)

func TestCurly(t *testing.T) {
	payload1 := `
curly  -i -X POST \
	http://localhost:8090/api/v1/products \
	-H "Content-Type:application/json" \
	-d {"name":"ugali"}
	`
	base := "http://localhost:8090/api/v1"
	sample := []struct {
		e     endpoint
		curly string
	}{
		{
			endpoint{
				Path:   "/products",
				Method: methodGet,
			}, "curly " + base + "/products",
		},
		{
			endpoint{
				Path:    "/products",
				Method:  methodPost,
				Payload: `{"name":"ugali"}`,
			}, payload1,
		},
		{
			endpoint{
				Path: "/products/:id",
				Params: []param{
					{
						Name:    "id",
						Type:    "ind64",
						Default: 1,
					},
				},
				Method: methodGet,
			}, "curly " + base + "/products/1",
		},
	}
	for _, v := range sample {
		e := strings.TrimSpace(v.e.curly(base))
		c := strings.TrimSpace(v.curly)
		if e != c {
			t.Errorf("expected %s got %s", c, e)
		}
	}
}
