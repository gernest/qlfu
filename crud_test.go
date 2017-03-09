package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cznic/ql"
)

func TestTemplates_create(t *testing.T) {
	createSample := []*field{
		{"id", 1},
		{"name", "gernest"},
		{"profession", "coder"},
	}
	data := make(map[string]interface{})
	data["model"] = "user"
	data["fields"] = createSample
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "create", data)
	if err != nil {
		t.Fatal(err)
	}
	expect := `
begin transaction;
  insert into user (id, name, profession)
  values ($1, $2, $3);
commit;
	`
	expect = strings.TrimSpace(expect)
	v := buf.String()
	v = strings.TrimSpace(v)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}

func TestTemplate_get_all(t *testing.T) {
	data := make(map[string]interface{})
	data["model"] = "user"
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "get_all", data)
	if err != nil {
		t.Fatal(err)
	}
	expect := "select * from user"
	expect = strings.TrimSpace(expect)
	v := buf.String()
	v = strings.TrimSpace(v)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}
func TestTemplates_update_last_id(t *testing.T) {
	data := make(map[string]interface{})
	data["model"] = "users"
	data["id"] = "id"
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "update_last_id", data)
	if err != nil {
		t.Fatal(err)
	}
	expect := `
begin transaction;
  update users id=$1 where id()=$1 ;
commit;
	`
	expect = strings.TrimSpace(expect)
	v := buf.String()
	v = strings.TrimSpace(v)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}

func TestCRUD_create(t *testing.T) {
	db, err := ql.OpenMem()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	_, _, err = db.Run(ql.NewRWCtx(), `
	begin transaction;
		create table user(
			id int64,
			name string,
		);
	commit;
	`)
	if err != nil {
		t.Fatal(err)
	}
	i, err := db.Info()
	if err != nil {
		t.Fatal(err)
	}
	s, err := schemaFromDBInfo(i)
	c := &crud{
		db:     db,
		schema: s,
	}
	props := make(modelProps)
	props["name"] = "gernest"
	p, err := c.create("user", props)
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := p["id"]; ok {
		n := id.(int64)
		if n != 1 {
			t.Errorf("expected 1 got %d", n)
		}
	} else {
		t.Errorf("id must be set")
	}
}
