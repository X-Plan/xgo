// term.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-09
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
		tms []term
		ts  = strings.Split(tag, ",")
		m   = make(map[string]string)
	)

	for _, t := range ts {
		// 空白串直接跳过.
		if isspace(t) {
			continue
		}
		kv := strings.SplitN(t, "=", 2)
		for i, _ := range kv {
			kv[i] = strings.TrimSpace(kv[i])
		}
		if _, ok := m[kv[0]]; ok {
			panic(fmt.Sprintf("%s: duplicate term '%s'", name, kv[0]))
		}

		if len(kv) == 1 {
			m[kv[0]] = ""
			tms = append(tms, newTerm(name, kv[0], ""))
		} else {
			m[kv[0]] = kv[1]
			tms = append(tms, newTerm(name, kv[0], kv[1]))
		}
	}
	return tms
}

func newTerm(name, k, v string) term {
	tm := term{name: name}
	switch k {
	case "noempty":
		if !isspace(v) {
			tm.panic("invalid term 'noempty=%s'", v)
		}
		tm.t, tm.check = tnoempty, tm.noempty
	case "min":
		tm.t, tm.check, tm.v = tmin, tm.template(tm.less), getValue(tmin, v, name)
	case "max":
		tm.t, tm.check, tm.v = tmax, tm.template(tm.greater), getValue(tmax, v, name)
	case "default":
		tm.t, tm.check, tm.v = tdefault, tm.template(tm.set), getValue(tdefault, v, name)
	case "match":
		tm.t, tm.check, tm.v = tmatch, tm.match, regexp.MustCompile(v)
	default:
	}
	return tm
}

func (tm term) noempty(v rft.Value) error {
	// 非空不能作用于bool类型, 因为这样产生的语
	// 义会使结果恒为真. 这样的选项没有任何意义.
	if v.Kind() == rft.Bool {
		tm.panic("bool type can'tm support 'noempty' term")
	}
	if tm.iszero(v) {
		return tm.errorf("is empty")
	}
	return nil
}

func (tm term) match(v rft.Value) error {
	if v.Kind() != rft.String {
		tm.panic("%v type can't support 'match' term", v.Kind())
	}
	if re := tm.v.(*regexp.Regexp); re.MatchString(v.String()) {
		return tm.errorf("'%s' not match 'match=%s' term", v.String(), re)
	}
	return nil
}

func (tm term) template(bop func(x rft.Value, y interface{}) bool) func(v rft.Value) error {
	return func(v rft.Value) error {
		var (
			ok bool
			tv interface{}
		)

		switch v.Kind() {
		case rft.Uint, rft.Uint8, rft.Uint16, rft.Uint32, rft.Uint64:
			ok = bop(v, tm.v)
		case rft.Int, rft.Int8, rft.Int16, rft.Int32, rft.Int64:
			switch tm.v.(type) {
			case uint64:
				tv = int64(tm.v.(uint64))
			case time.Duration:
				tv = int64(tm.v.(time.Duration))
			}
			ok = bop(v, tv)
		case rft.Float32, rft.Float64:
			switch tm.v.(type) {
			case uint64:
				tv = float64(tm.v.(uint64))
			case int64:
				tv = float64(tm.v.(int64))
			}
			ok = bop(v, tv)
		case rft.Bool, rft.String:
			if tm.t == tdefault {
				ok = bop(v, tm.v)
				break
			}
			fallthrough
		default:
			tm.panic("%v type can't support '%s' term", v.Kind(), tm.t)
		}

		if !ok {
			return tm.errorf("can't satisfy term '%s=%v'", tm.t, tm.v)
		}
		return nil
	}
}

func (tm term) less(x rft.Value, y interface{}) bool {
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

func (tm term) greater(x rft.Value, y interface{}) bool {
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

func (tm term) set(x rft.Value, y interface{}) bool {
	if tm.iszero(x) {
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

func (tm term) iszero(v rft.Value) bool {
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
			z = z && tm.iszero(v.Index(i))
		}
	case rft.Struct:
		for i := 0; i < v.NumField(); i++ {
			z = z && tm.iszero(v.Field(i))
		}
	default:
		// bool, int, uint, float, string
		z = (v.Interface() == rft.Zero(v.Type()).Interface())
	}

	return z
}

func (tm term) panic(format string, args ...interface{}) {
	panic(fmt.Sprintf(tm.name+": "+format, args...))
}

func (tm term) errorf(format string, args ...interface{}) error {
	return fmt.Errorf(tm.name+": "+format, args...)
}

func isspace(s string) bool {
	return regexp.MustCompile(`^[[:space:]]*$`).MatchString(s)
}

func isbool(s string) bool {
	return regexp.MustCompile(`^(true|True|TRUE|false|False|FALSE)$`).MatchString(s)
}

func getValue(t termtype, v string, name string) interface{} {
	// 标准库https://golang.org/pkg/strconv/#ParseBool
	// 会对0,1进行解释, 这不将这两者看作是bool类型.
	if isbool(v) {
		if t != tdefault {
			goto panic_exit
		}
		b, _ := strconv.ParseBool(v)
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

panic_exit:
	panic(fmt.Sprintf("%s: invalid term '%s=%s'", name, t, v))
	return nil
}
