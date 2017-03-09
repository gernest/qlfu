package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"

	"bytes"
	"text/tabwriter"

	"time"

	"github.com/cznic/ql"
	"github.com/jinzhu/inflection"
)

func special(src string) bool {
	return strings.HasPrefix(src, "__")
}

type dbSchema struct {
	tables map[string]*table
}

func schemaFromJSON(src io.Reader) (*dbSchema, error) {
	o := make(map[string]interface{})
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &o)
	if err != nil {
		return nil, err
	}
	s := &dbSchema{tables: make(map[string]*table)}
	for k, v := range o {
		switch rv := v.(type) {
		case map[string]interface{}:
			t := &table{name: tableName(k)}
			for nk, nv := range rv {
				c := &column{name: nk}
				switch nv.(type) {
				case bool:
					c.typ = ql.Bool
				case float64:
					c.typ = ql.Float64
				case string:
					if _, ok := toTime(nv.(string)); ok {
						c.typ = ql.Time
					} else {
						c.typ = ql.String
					}
				case nil:
					return nil, fmt.Errorf("%s.%v : null objects not supported",
						k, nk,
					)
				case []string:
					return nil, fmt.Errorf("%s.%v : array objects not supported",
						k, nk,
					)
				case map[string]interface{}:
					relTable := &table{name: tableName(nk)}
					for relKey, relVal := range nv.(map[string]interface{}) {
						rc := &column{name: relKey}
						switch relVal.(type) {
						case bool:
							rc.typ = ql.Bool
						case float64:
							rc.typ = ql.Float64
						case string:
							if _, ok := toTime(relVal.(string)); ok {
								rc.typ = ql.Time
							} else {
								rc.typ = ql.String
							}
						case nil:
							return nil, fmt.Errorf("%s.%v : null objects not supported",
								k, nk,
							)
						case []string:
							return nil, fmt.Errorf("%s.%v : array objects not supported",
								k, nk,
							)
						default:
							return nil, fmt.Errorf("%s.%v : fishy type uh",
								k, nk,
							)
						}
						relTable.columns = append(relTable.columns, rc)
					}
					t.hasOne = &relation{
						srcCol:    c.name,
						destTable: relTable.name,
					}
					t.related = true
					s.tables[relTable.name] = relTable
				}
				t.columns = append(t.columns, c)
			}
			s.tables[t.name] = t
		default:
			return nil, errors.New("top level values must be valid objects")
		}
	}
	err = s.prepareRelations()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func toTime(src string) (time.Time, bool) {
	t, err := time.Parse(time.ANSIC, src)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func schemaFromDBInfo(i *ql.DbInfo) (*dbSchema, error) {
	s := &dbSchema{tables: make(map[string]*table)}
	for _, v := range i.Tables {
		if special(v.Name) {
			continue
		}
		tb := &table{name: v.Name}
		tb.i = &v
		for _, cols := range v.Columns {
			c := &column{
				name: cols.Name,
				i:    &cols,
				typ:  cols.Type,
			}
			tb.columns = append(tb.columns, c)
		}
		s.tables[tb.name] = tb
	}
	return buildRelation(s), nil
}

func buildRelation(s *dbSchema) *dbSchema {
	t := make(map[string]*table)
	for k, v := range s.tables {
		for _, cols := range v.columns {
			if foreignKey(cols.name) {
				ft := foreignKeyTable(cols.name)
				if _, ok := s.tables[ft]; ok {
					v.related = true
					v.hasOne = &relation{
						srcCol:    cols.name,
						destTable: ft,
					}
				}
			}
		}
		t[k] = v
	}
	s.tables = t
	return s
}

func foreignKey(src string) bool {
	return strings.HasSuffix(src, "_id")
}

func foreignKeyTable(src string) string {
	return strings.TrimSuffix(src, "_id")
}

func (d *dbSchema) prepareRelations() error {
	s := make(map[string]*table)
	for k, v := range d.tables {
		if v.related {
			if v.hasOne != nil {
				fk := v.hasOne.destTable + "_id"
				c := &column{name: fk, typ: ql.Int64}
				if dt, ok := d.tables[v.hasOne.destTable]; ok {
					if idx, ok := dt.colID("id"); ok {
						c.typ = dt.columns[idx].typ
					}
				} else {
					return fmt.Errorf("missing relation %s", v.hasOne.destTable)
				}
				if idx, ok := v.colID(v.hasOne.srcCol); ok {
					v.columns[idx].typ = c.typ
					n := v.hasOne.destTable + "_id"
					v.columns[idx].name = n
					v.hasOne.srcCol = n
				}
				v.columns = append(v.columns)
			}
		}
		v.prepare()
		s[k] = v
	}
	d.tables = s
	return nil
}

func (d *dbSchema) migration(i int) (sql string) {
	var tbs tableList
	sql = fmt.Sprintln("begin transaction;")
	for _, v := range d.tables {
		tbs = append(tbs, v)
	}
	sort.Sort(tbs)
	for _, v := range tbs {
		sort.Sort(v.columns)
		sql += fmt.Sprintln(v.migration(i + 2))
	}
	sql += fmt.Sprintln("commit;")
	return
}

type table struct {
	i          *ql.TableInfo
	name       string
	related    bool
	hasOne     *relation
	hasMany    *relation
	manyToMany *relation
	columns    columnList
}

func (t *table) prepare() {
	var hasID bool
	for i := 0; i < len(t.columns)-1; i++ {
		c := t.columns[i]
		if c.name == "id" {
			hasID = true
		}
	}
	if !hasID {
		t.columns = append(t.columns, &column{name: "id", typ: ql.Int64})
	}
}

func tableName(name string) string {
	return inflection.Plural(name)
}

// i is the level of indentation
func (t *table) migration(i int) (sql string) {
	sql += fmt.Sprintf("%s create table %s (\n", indent(i), t.name)
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	for k, v := range t.columns {
		if k == 0 {
			fmt.Fprintf(w, "%s%s\t%s", indent(i+2), v.name, v.typ)
		} else {
			fmt.Fprintf(w, ",\n%s%s\t%s", indent(i+2), v.name, v.typ)
		}
	}
	_ = w.Flush()
	sql += buf.String()
	sql += fmt.Sprint(");")
	return
}

func indent(n int) string {
	a := ""
	for i := 0; i < n; i++ {
		a += " "
	}
	return a
}
func (t *table) colID(name string) (int, bool) {
	for k, v := range t.columns {
		if v.name == name {
			return k, true
		}
	}
	return 0, false
}

type column struct {
	i            *ql.ColumnInfo
	name         string
	typ          ql.Type
	val          reflect.Value
	defaultValue interface{}
}

type columnList []*column

func (c columnList) Len() int           { return len(c) }
func (c columnList) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c columnList) Less(i, j int) bool { return c[i].typ.String() < c[j].typ.String() }

type tableList []*table

func (c tableList) Len() int           { return len(c) }
func (c tableList) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c tableList) Less(i, j int) bool { return c[i].name < c[j].name }

func tableFromMap(v map[string]interface{}) (*table, error) {
	return nil, nil
}

type relation struct {
	srcCol    string
	destTable string
}

// This is the only comment in this project.
//
// Thank you for taking your time to read this code base. I wanted to write a working application
// that can easily be understood without any comment or extra story lines. For some extent I believe
// I succeeded.
//
// go is beautiful, I feel sad that I have to move away from it and start all over again maybe
// hopefully fall in love with another language.
//
// There is so much good and interesting bits in the standard library and in the go community.
// If you are like me, I mean if you prefer not to reinvent wheels please take some time to admire
// what Go has for you.
//
// With Regards
// gernest
