// term.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-10
package xvalid

import (
	"fmt"
	rft "reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	tdefault termtype = iota
	tnoempty
	tmin
	tmax
	tmatch
)

type termtype int

var termstr = []string{"tdefault", "noempty", "min", "max", "match"}

func (tt termtype) String() string {
	return termstr[int(tt)]
}

type terms []*term

// 检测tms中的条件是否存在矛盾, 如果存在则panic.
//  当前版本下会执行以下操作:
// 1. 如果同时存在default和min(max), default的的值
// 必须满足min(max)的限制.
func (tms terms) conflict() {
	var (
		err           error
		def, min, max *term
	)

	for _, tm := range tms {
		switch tm.t {
		case tdefault:
			def = tm
		case tmin:
			min = tm
		case tmax:
			min = tm
		}
	}

	if def != nil {
		if min != nil {
			if err = min.check(rft.ValueOf(def.v)); err != nil {
				max.panic("term '%s' and term '%s' are contradictory", def, min)
			}
		}
		if max != nil {
			if err = max.check(rft.ValueOf(def.v)); err != nil {
				max.panic("term '%s' and term '%s' are contradictory", def, max)
			}
		}
	}
}

// 重排tms中项的顺序, 当前版本执行的操作有:
// 1. default会被放在优先处理的位置. 优先处理该项的
// 原因在于保证后续的检测条件是在预设值的基础上进行
// 的. 比如:noempty,default=10. 当未设置该值时应该
// 先进行
func (tms *terms) resort() {
	sort.Sort(tms)
}

func (tms *terms) Len() int {
	return len(*tms)
}

func (tms *terms) Less(i, j int) bool {
	return (*tms)[i].t < (*tms)[j].t
}

func (tms *terms) Swap(i, j int) {
	(*tms)[i], (*tms)[j] = (*tms)[j], (*tms)[i]
}

func newTerms(name, tag string) terms {
	var (
		tms []*term
		ts  = strings.Split(strings.TrimSpace(tag), ",")
		m   = make(map[string]string)
	)

	for _, t := range ts {
		var (
			pair        = strings.SplitN(strings.TrimSpace(t), "=", 2)
			k, v string = strings.TrimSpace(pair[0]), ""
		)

		if len(pair) == 1 {
			if isspace(k) {
				continue
			}
		} else if len(pair) == 2 {
			v = strings.TrimSpace(pair[1])
		}

		if _, ok := m[k]; ok {
			panic(fmt.Sprintf("%s: duplicate term '%s'", name, k))
		}
		m[k] = v
		tms = append(tms, newTerm(name, k, v))
	}
	return tms
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

func newTerm(name, k, v string) *term {
	tm := &term{name: name}
	switch k {
	case "noempty":
		if !isspace(v) {
			tm.panic("invalid term 'noempty=%s'", v)
		}
		tm.t, tm.check = tnoempty, tm.noempty
	case "min":
		tm.t, tm.v, tm.check = tmin, getValue(tmin, v, name), tm.template(tm.greater)
	case "max":
		tm.t, tm.v, tm.check = tmax, getValue(tmax, v, name), tm.template(tm.less)
	case "default":
		tm.t, tm.v, tm.check = tdefault, getValue(tdefault, v, name), tm.template(tm.set)
	case "match":
		if len(v) < 2 || v[0] != '/' || v[len(v)-1] != '/' {
			tm.panic("invalid term 'match=%s'", v)
		}
		tm.t, tm.v, tm.check = tmatch, regexp.MustCompile(v[1:len(v)-1]), tm.match
	default:
		tm.panic("unknown term '%s'", k)
	}
	return tm
}

func (tm *term) noempty(v rft.Value) error {
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

func (tm *term) match(v rft.Value) error {
	if v.Kind() != rft.String {
		tm.panic("%v type can't support 'match' term", v.Kind())
	}
	if re := tm.v.(*regexp.Regexp); !re.MatchString(v.String()) {
		return tm.errorf("'%s' not match '%s'", v.String(), re)
	}
	return nil
}

func (tm *term) template(bop func(x rft.Value, y interface{}) bool) func(v rft.Value) error {
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
			case int64:
				tv = tm.v.(int64)
			case uint64:
				tv = int64(tm.v.(uint64))
			case time.Duration:
				tv = int64(tm.v.(time.Duration))
			}
			ok = bop(v, tv)
		case rft.Float32, rft.Float64:
			switch tm.v.(type) {
			case float64:
				tv = tm.v.(float64)
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
			return tm.errorf("can't satisfy term '%s'", tm)
		}
		return nil
	}
}

func (tm *term) less(x rft.Value, y interface{}) bool {
	var ok bool
	switch y.(type) {
	case uint64:
		ok = (x.Uint() <= y.(uint64))
	case int64:
		ok = (x.Int() <= y.(int64))
	case float64:
		ok = (x.Float() <= y.(float64))
	}
	return ok
}

func (tm *term) greater(x rft.Value, y interface{}) bool {
	var ok bool
	switch y.(type) {
	case uint64:
		ok = (x.Uint() >= y.(uint64))
	case int64:
		ok = (x.Int() >= y.(int64))
	case float64:
		ok = (x.Float() >= y.(float64))
	}
	return ok
}

func (tm *term) set(x rft.Value, y interface{}) bool {
	if tm.iszero(x) {
		switch y.(type) {
		case bool:
			x.SetBool(y.(bool))
		case int64:
			x.SetInt(y.(int64))
		case uint64:
			x.SetUint(y.(uint64))
		case float64:
			x.SetFloat(y.(float64))
		case string:
			x.SetString(y.(string))
		}
	}
	return true
}

func (tm *term) iszero(v rft.Value) bool {
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

func (tm *term) panic(format string, args ...interface{}) {
	panic(fmt.Sprintf(tm.name+": "+format, args...))
}

func (tm *term) errorf(format string, args ...interface{}) error {
	return fmt.Errorf(tm.name+": "+format, args...)
}

func (tm *term) String() string {
	return fmt.Sprintf("%s=%v", tm.t, tm.v)
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
