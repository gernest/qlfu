package main

import (
	"crypto/rand"
	"encoding/json"
	"net/http"

	"crypto/md5"

	"path/filepath"

	"fmt"

	"os"

	"strings"

	"github.com/cznic/lldb"
	"github.com/cznic/ql"
	"github.com/gernest/alien"
)

type api struct {
	r       *alien.Mux
	c       *crud
	db      *ql.DB
	dba     dba
	service *service
	baseURL string
}

func newAPI(dir, baseURL string) (*api, error) {
	a := &api{}
	s := newSdba(dir)
	db, err := s.current()
	if err != nil {
		return nil, err
	}
	c, err := newCrud(db)
	a.c = c
	a.db = db
	a.dba = s
	a.baseURL = baseURL
	err = a.load()
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *api) load() error {
	r := alien.New()
	_ = r.Get("/schema", a.schema)
	_ = r.Post("/schema", a.newSchema)
	a.r = r
	return a.handleService()
}

func (a *api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.r.ServeHTTP(w, r)
}

func (a *api) schema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plaint")
	_, err := a.c.writeSchema(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *api) newSchema(w http.ResponseWriter, r *http.Request) {
	s, err := schemaFromJSON(r.Body)
	if err != nil {
		jsonErr(w, err, http.StatusBadRequest)
		return
	}
	db, err := a.dba.fresh()
	if err != nil {
		jsonErr(w, err, http.StatusBadRequest)
		return
	}
	err = runMigration(db, s)
	if err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}
	c, err := newCrud(db)
	if err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}
	a.c = c
	err = a.load()
	if err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}
	jsonOk(w)
}

func jsonErr(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	d := make(map[string]interface{})
	d["error"] = err.Error()
	d["message"] = http.StatusText(code)
	b, _ := json.Marshal(d)
	_, _ = w.Write(b)
	w.Header().Set("Content-Type", "application/json")
}

func jsonOk(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	d := make(map[string]interface{})
	d["status"] = http.StatusText(http.StatusOK)
	b, _ := json.Marshal(d)
	_, _ = w.Write(b)
	w.Header().Set("Content-Type", "application/json")
}

func jsonRes(w http.ResponseWriter, d interface{}) {
	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(d)
	_, _ = w.Write(b)
	w.Header().Set("Content-Type", "application/json")
}

func runMigration(db *ql.DB, s *dbSchema) error {
	m := s.migration(0)
	_, _, err := db.Run(ql.NewRWCtx(), m)
	return err
}

type dba interface {
	current() (*ql.DB, error)
	fresh() (*ql.DB, error)
	close() error
}

type sdba struct {
	dir string
	c   *ql.DB
}

func newSdba(dir string) *sdba {
	return &sdba{dir: dir}
}

func (s *sdba) current() (*ql.DB, error) {
	if s.c != nil {
		return s.c, nil
	}
	return s.fresh()
}

func (s *sdba) fresh() (*ql.DB, error) {
	b := make([]byte, 15)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	m := fmt.Sprintf("%x", md5.Sum(b))
	f := filepath.Join(s.dir, m)
	q, err := ql.OpenFile(f, &ql.Options{
		CanCreate: true,
		TempFile:  s.tmpFile,
	})
	s.c = q
	return s.c, nil
}

func (s *sdba) tmpFile(dir, prefix string) (lldb.OSFile, error) {
	b := make([]byte, 15)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	m := fmt.Sprintf("%x", md5.Sum(b))
	f := filepath.Join(s.dir, prefix, m)
	return os.Open(f)
}

func (s *sdba) close() error {
	return os.RemoveAll(s.dir)
}

type endpoint struct {
	Path    string  `json:"path"`
	Params  []param `json:"params"`
	Method  string  `json:"method"`
	Payload string  `json:"payload"`
	handler http.HandlerFunc
}

const curlyGet = `
curly %s
`
const curlyPost = `
curly  -i -X POST \
	%s \
	-H "Content-Type:application/json" \
	-d %s
`

func (e endpoint) curly(base string) string {
	p := e.Path
	switch e.Method {
	case methodGet:
		if e.Params != nil {
			p = replaceParams(p, e.Params)
		}
		return fmt.Sprintf(curlyGet, base+p)
	case methodPost:
		if e.Params != nil {
			p = replaceParams(p, e.Params)
		}
		return fmt.Sprintf(curlyPost, base+p, e.Payload)
	}
	return ""
}

func replaceParams(p string, params []param) string {
	for _, v := range params {
		p = strings.Replace(p, ":"+v.Name, fmt.Sprint(v.Default), -1)
	}
	return p
}

type param struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Desc    string      `json:"desc"`
	Default interface{} `json:"default,omitempty"`
}

type service struct {
	Version   string `json:"version"`
	Endpoints []endpoint
}

func (a *api) handleService() error {
	s, err := a.c.service()
	if err != nil {
		return err
	}
	a.service = s
	return a.registerService()
}

func (a *api) registerService() error {
	curl := a.r.Group(fmt.Sprintf("/curly/v%s", a.service.Version))
	e := a.r.Group(fmt.Sprintf("/v%s", a.service.Version))
	for _, point := range a.service.Endpoints {
		switch point.Method {
		case methodGet:
			_ = curl.Get(point.Path, curlHandler(a.baseURL, point))
			_ = e.Get(point.Path, point.handler)
		case methodPost:
			_ = curl.Post(point.Path, curlHandler(a.baseURL, point))
			_ = e.Post(point.Path, point.handler)
		case methodPut:
			_ = curl.Put(point.Path, curlHandler(a.baseURL, point))
			_ = e.Put(point.Path, point.handler)
		}
	}
	_ = a.r.Get(fmt.Sprintf("/v%s", a.service.Version), a.showService)
	return nil
}

func (a *api) showService(w http.ResponseWriter, r *http.Request) {
	jsonRes(w, a.service)
}

func curlHandler(base string, e endpoint) func(http.ResponseWriter, *http.Request) {
	txt := e.curly(base)
	return func(w http.ResponseWriter, r *http.Request) {
		textOk(w, txt)
	}
}

func textOk(w http.ResponseWriter, txt string) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(txt))
}
