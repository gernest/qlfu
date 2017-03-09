package main

import (
	"io/ioutil"
	"testing"

	"strings"

	"github.com/cznic/ql"
)

func TestSchema_fromDBInfo(t *testing.T) {
	b, err := ioutil.ReadFile("schema.ql")
	if err != nil {
		t.Fatal(err)
	}
	sample := string(b)
	db, err := ql.OpenMem()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = db.Run(ql.NewRWCtx(), sample)
	if err != nil {
		t.Fatal(err)
	}
	i, err := db.Info()
	if err != nil {
		t.Fatal(err)
	}
	ds, err := schemaFromDBInfo(i)
	if err != nil {
		t.Fatal(err)
	}
	s := ds.migration(0)
	if s != sample {
		t.Errorf("expected \n%s\nGot\n%s", sample, s)
	}
	//ioutil.WriteFile("schema.ql", []byte(s), 0600)
}

func TestSchemaFromJSON(t *testing.T) {
	src := `
{
   "product":{
      "name":"Maize",
      "user":{
         "username":"gernest",
         "email":"gernest@example.com"
      }
   }
}
`
	s, err := schemaFromJSON(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	expect := `
begin transaction;
   create table products (
    users_id int64,
    id       int64,
    name     string);
   create table users (
    id       int64,
    username string,
    email    string);
commit;
	`
	expect = strings.TrimSpace(expect)
	v := s.migration(0)
	v = strings.TrimSpace(v)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}

func TestSchema_handle_time(t *testing.T) {
	src := `
{
   "product":{
      "name":"Maize",
      "user":{
         "username":"gernest",
         "email":"gernest@example.com",
         "created_at":"Mon Jan 2 15:04:05 2006",
         "updated_at":"Mon Jan 2 15:04:05 2006"
      },
      "created_at":"Mon Jan 2 15:04:05 2006",
      "updated_at":"Mon Jan 2 15:04:05 2006"
   }
}
`
	s, err := schemaFromJSON(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	expect := `
begin transaction;
   create table products (
    users_id   int64,
    id         int64,
    name       string,
    created_at time,
    updated_at time);
   create table users (
    id         int64,
    username   string,
    email      string,
    created_at time,
    updated_at time);
commit;
	`
	expect = strings.TrimSpace(expect)
	v := s.migration(0)
	v = strings.TrimSpace(v)
	if v != expect {
		t.Errorf("expected %s got %s", expect, v)
	}
}
