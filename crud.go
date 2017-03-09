package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"fmt"

	"io"

	"io/ioutil"

	"sort"

	"time"

	"github.com/cznic/ql"
	"github.com/gernest/alien"
	"github.com/jinzhu/inflection"
)

var tpl *template.Template

const activeVersion = "1"
const methodGet = "get"
const methodPost = "post"
const methodPut = "put"

func init() {
	funcs := make(template.FuncMap)
	funcs["incr"] = func(i int) int {
		return i + 1
	}
	t, err := template.New("qlfu").Funcs(funcs).Parse(qlfuTpl)
	if err != nil {
		log.Fatal(err)
	}
	tpl = t
}

type crud struct {
	db     *ql.DB
	mu     sync.Mutex
	schema *dbSchema
}

func newCrud(db *ql.DB) (*crud, error) {
	c := &crud{db: db}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

type field struct {
	Name  string
	value interface{}
}

type modelProps map[string]interface{}

func (m modelProps) propProperty(name string) (modelProps, bool) {
	if o, ok := m[name]; ok {
		ov, ok := o.(map[string]interface{})
		if ok {
			return modelProps(ov), true
		}
	}
	return nil, false
}
func (c *crud) create(model string, props modelProps) (modelProps, error) {
	var f []*field
	t, ok := c.schema.tables[model]
	if !ok {
		return nil, fmt.Errorf("model %s not found", model)
	}
	if t.related {
		if t.hasOne != nil {
			if one, ok := c.findHasOneProps(t.hasOne.destTable, props); ok {
				rp, err := c.create(t.hasOne.destTable, one)
				if err != nil {
					return nil, err
				}
				fk := t.hasOne.destTable + "_id"
				props[fk] = rp["id"]
			}
		}
	}
	for k, v := range props {
		if _, ok := t.colID(k); ok {
			f = append(f, &field{
				Name:  k,
				value: v,
			})
		}
	}
	ctx := make(map[string]interface{})
	ctx["model"] = model
	ctx["fields"] = f
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "create", ctx)
	if err != nil {
		return nil, err
	}
	var v []interface{}
	for _, fv := range f {
		v = append(v, fv.value)
	}
	qctx := ql.NewRWCtx()
	_, _, err = c.db.Run(qctx, buf.String(), v...)
	if err != nil {

		return nil, err
	}

	nctx := make(map[string]interface{})
	nctx["id"] = "id"
	nctx["model"] = model

	buf.Reset()
	err = tpl.ExecuteTemplate(&buf, "update_last_id", nctx)
	if err != nil {
		return nil, err
	}

	_, _, err = c.db.Run(ql.NewRWCtx(), buf.String(), qctx.LastInsertID)
	if err != nil {
		return nil, err
	}
	props["id"] = qctx.LastInsertID
	return props, nil
}

func (c *crud) findHasOneProps(model string, props modelProps) (modelProps, bool) {
	if o, ok := props.propProperty(model); ok {
		return o, true
	}
	return props.propProperty(inflection.Singular(model))
}

func (c *crud) getAll(model string) ([]modelProps, error) {
	ctx := make(map[string]interface{})
	ctx["model"] = model
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "get_all", ctx)
	if err != nil {
		return nil, err
	}
	rs, _, err := c.db.Run(ql.NewRWCtx(), buf.String())
	if err != nil {
		return nil, err
	}
	var o []modelProps
	for _, v := range rs {
		names, err := v.Fields()
		if err != nil {
			return nil, err
		}
		_ = v.Do(false, func(data []interface{}) (bool, error) {
			p := make(modelProps)
			for k, value := range data {
				p[names[k]] = value
			}
			o = append(o, p)
			return true, nil
		})
	}

	return o, nil
}

func (c *crud) getByID(model string, id int64) ([]modelProps, error) {
	ctx := make(map[string]interface{})
	ctx["model"] = model
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "get_by_id", ctx)
	if err != nil {
		return nil, err
	}
	rs, _, err := c.db.Run(ql.NewRWCtx(), buf.String(), id)
	if err != nil {
		return nil, err
	}
	var o []modelProps
	for _, v := range rs {
		names, err := v.Fields()
		if err != nil {
			return nil, err
		}
		_ = v.Do(false, func(data []interface{}) (bool, error) {
			p := make(modelProps)
			for k, value := range data {
				p[names[k]] = value
			}
			o = append(o, p)
			return true, nil
		})
	}
	return o, nil
}

const qlfuTpl = `
{{define "create"}}
begin transaction;
  insert into {{.model}} ({{range $k,$v:=.fields}}{{if eq $k 0}}{{$v.Name}}{{else}}, {{$v.Name}}{{end}}{{end}})
  values ({{range $k,$v:=.fields}}{{if eq $k 0}}${{incr $k}}{{else}}, ${{incr $k}}{{end}}{{end}});
commit;
{{end}}
{{define "update_last_id"}}
begin transaction;
  update {{.model}} {{.id}}=$1 where id()=$1 ;
commit;
{{end}}
{{define "get_by_id"}}
  select * from {{.model}} where id=$1;
{{end}}
{{define "get_all"}}
  select * from {{.model}}
{{end}}
`

func (c *crud) writeSchema(w io.Writer) (int, error) {
	return fmt.Fprint(w, c.schema.migration(0))
}

func (c *crud) load() error {
	i, err := c.db.Info()
	if err != nil {
		return err
	}
	s, err := schemaFromDBInfo(i)
	if err != nil {
		return err
	}
	c.schema = s
	return nil
}

func (c *crud) service() (*service, error) {
	s := &service{
		Version: activeVersion,
	}
	for _, m := range c.models() {
		s.Endpoints = append(s.Endpoints, endpoint{
			Path:    "/" + m.name,
			Method:  methodPost,
			Payload: samplePayload(m, true),
			handler: c.createHandler(m.name),
		})
		s.Endpoints = append(s.Endpoints, endpoint{
			Path:    "/" + m.name,
			Method:  methodGet,
			handler: c.getAllHandler(m.name),
		})
		s.Endpoints = append(s.Endpoints, endpoint{
			Path:   "/" + m.name + "/:id",
			Method: methodGet,
			Params: []param{
				{
					Name:    "id",
					Type:    "int64",
					Desc:    "the id of " + m.name + " object",
					Default: 1,
				},
			},
			handler: c.getByIDHandler(m.name),
		})
	}
	return s, nil
}

func (c *crud) models() tableList {
	var l tableList
	for _, t := range c.schema.tables {
		l = append(l, t)
	}
	sort.Sort(l)
	return l
}

func (c *crud) createHandler(model string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		prop := make(modelProps)
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			jsonErr(w, err, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(b, &prop)
		if err != nil {
			jsonErr(w, err, http.StatusInternalServerError)
			return
		}
		o, err := c.create(model, prop)
		if err != nil {
			jsonErr(w, err, http.StatusInternalServerError)
			return
		}
		jsonRes(w, o)
	}
}

func (c *crud) getAllHandler(model string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		o, err := c.getAll(model)
		if err != nil {
			jsonErr(w, err, http.StatusInternalServerError)
			return
		}
		if o == nil {
			jsonErr(w, errors.New("no records found"), http.StatusNotFound)
			return
		}
		jsonRes(w, o)
	}
}
func (c *crud) getByIDHandler(model string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p := alien.GetParams(r)
		id := p.Get("id")
		i, err := strconv.Atoi(id)
		if err != nil {
			jsonErr(w, err, http.StatusInternalServerError)
			return
		}
		o, err := c.getByID(model, int64(i))
		if err != nil {
			jsonErr(w, err, http.StatusInternalServerError)
			return
		}
		if o == nil {
			jsonErr(w, errors.New("no records found"), http.StatusNotFound)
			return
		}
		jsonRes(w, o)
	}
}

func samplePayload(t *table, omitID bool) string {
	o := make(modelProps)
	for _, c := range t.columns {
		if c.name == "id" {
			if omitID {
				continue
			}
		}
		o[c.name] = sampleValue(c.name, c.typ)
	}
	b, _ := json.Marshal(o)
	return string(b)
}

func sampleValue(name string, typ ql.Type) interface{} {
	var v interface{}
	switch typ {
	case ql.String:
		v = name
	case ql.Int64:
		v = 1
	case ql.Time:
		v = time.Now().Format(time.ANSIC)
	}
	return v
}
