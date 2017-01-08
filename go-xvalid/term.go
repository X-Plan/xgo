// term.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-08
package xvalid

import (
	"fmt"
	rft "reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	tnoempty termtype = iota
	tmin
	tmax
	tdefault
	tmatch
)

type termtype int

var termstr = []string{"noempty", "min", "max", "default", "match"}

func (tt termtype) String() string {
	return termstr[int(tt)]
}

// term支持的类型有: bool, int类, uint类(除了uintptr),
// float类, ptr, string, array, slice, map, struct, interface.
// 其它类型会导致term相关的操作panic.
type term struct {
	t     termtype
	v     interface{}
	name  string
	check func(rft.Value) error
}

func newTerms(name, tag string) []term {
	var (
		terms []term
		ts    = strings.Split(tag, ",")
		m     = make(map[string]string)
	)

	for _, t := range ts {
		kv := strings.SplitN(t, "=", 2)
		if _, ok := m[kv[0]]; ok {
			panic(fmt.Sprintf("%s: duplicate term '%s'", name, kv[0]))
		}
		m[kv[0]] = kv[1]
		terms = append(terms, newTerm(name, kv[0], kv[1]))
	}
	return terms
}

func newTerm(name, k, v string) term {
	t := term{name: name}
	switch k {
	case "noempty":
		if !isspace(v) {
			t.panic("invalid term 'noempty=%s'", v)
		}
		t.t, t.check = tnoempty, t.noempty
	case "min":
		t.t, t.check, t.v = tmin, t.template(t.less), getValue(t.t, v, name)
	case "max":
		t.t, t.check, t.v = tmax, t.template(t.greater), getValue(t.t, v, name)
	case "default":
		t.t, t.check, t.v = tmax, t.template(t.set), getValue(t.t, v, name)
	case "match":
		t.t, t.check, t.v = tmatch, t.match, regexp.MustCompile(v)
	default:
	}
	return t
}

func (t term) noempty(v rft.Value) error {
	// 非空不能作用于bool类型, 因为这样产生的语
	// 义会使结果恒为真. 这样的选项没有任何意义.
	if v.Kind() == rft.Bool {
		t.panic("bool type can't support 'noempty' term")
	}
	if t.iszero(v) {
		return t.errorf("is empty")
	}
	return nil
}

func (t term) match(v rft.Value) error {
	if v.Kind() != rft.String {
		t.panic("%v type can't support 'match' term", v.Kind())
	}
	if re := t.v.(*regexp.Regexp); re.MatchString(v.String()) {
		return t.errorf("'%s' not match 'match=%s' term", v.String(), re)
	}
	return nil
}

func (t term) template(bop func(x rft.Value, y interface{}) bool) func(v rft.Value) error {
	return func(v rft.Value) error {
		var (
			ok bool
			tv interface{}
		)

		switch v.Kind() {
		case rft.Uint, rft.Uint8, rft.Uint16, rft.Uint32, rft.Uint64:
			ok = bop(v, t.v)
		case rft.Int, rft.Int8, rft.Int16, rft.Int32, rft.Int64:
			switch t.v.(type) {
			case uint64:
				tv = int64(t.v.(uint64))
			case time.Duration:
				tv = int64(t.v.(time.Duration))
			}
			ok = bop(v, tv)
		case rft.Float32, rft.Float64:
			switch t.v.(type) {
			case uint64:
				tv = float64(t.v.(uint64))
			case int64:
				tv = float64(t.v.(int64))
			}
			ok = bop(v, tv)
		case rft.Bool, rft.String:
			if t.t == tdefault {
				ok = bop(v, t.v)
				break
			}
			fallthrough
		default:
			t.panic("%v type can't support '%s' term", v.Kind(), t.t)
		}

		if !ok {
			return t.errorf("can't satisfy term '%s=%v'", t.t, t.v)
		}
		return nil
	}
}

func (t term) less(x rft.Value, y interface{}) bool {
	var ok bool
	switch y.(type) {
	case uint64:
		ok = (x.Uint() < y.(uint64))
	case int64:
		ok = (x.Int() < y.(int64))
	case float64:
		ok = (x.Float() < y.(float64))
	}
	return ok
}

func (t term) greater(x rft.Value, y interface{}) bool {
	var ok bool
	switch y.(type) {
	case uint64:
		ok = (x.Uint() > y.(uint64))
	case int64:
		ok = (x.Int() > y.(int64))
	case float64:
		ok = (x.Float() > y.(float64))
	}
	return ok
}

func (t term) set(x rft.Value, y interface{}) bool {
	if t.iszero(x) {
		switch y.(type) {
		case bool:
			x.SetBool(y.(bool))
		case uint64:
			x.SetInt(y.(int64))
		case int64:
			x.SetUint(y.(uint64))
		case float64:
			x.SetFloat(y.(float64))
		case string:
			x.SetString(y.(string))
		}
	}
	return true
}

func (t term) iszero(v rft.Value) bool {
	var (
		z = true
	)

	switch v.Kind() {
	case rft.Map, rft.Slice:
		return v.Len() == 0
	case rft.Interface, rft.Ptr:
		return v.IsNil()
	case rft.Array:
		for i := 0; i < v.Len(); i++ {
			z = z && t.iszero(v.Field(i))
		}
	case rft.Struct:
		for i := 0; i < v.NumField(); i++ {
			z = z && t.iszero(v.Field(i))
		}
	default:
		// bool, int, uint, float, string
		z = (v == rft.Zero(v.Type()))
	}

	return z
}

func (t term) panic(format string, args ...interface{}) {
	panic(fmt.Sprintf(t.name+": "+format, args...))
}

func (t term) errorf(format string, args ...interface{}) error {
	return fmt.Errorf(t.name+": "+format, args...)
}

func isspace(s string) bool {
	return regexp.MustCompile(`\s*`).MatchString(s)
}

func getValue(t termtype, v string, name string) interface{} {
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	} else if ui, err := strconv.ParseUint(v, 10, 64); err == nil {
		return ui
	} else if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	} else if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	} else if d, err := time.ParseDuration(v); err == nil {
		return d
	} else if t == tdefault {
		return v
	}
	panic(fmt.Sprintf("%s: invalid term '%s=%s'", name, t, v))
	return nil
}
