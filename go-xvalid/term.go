// term.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-14
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
	tidefault
	tinoempty
	timin
	timax
	timatch
)

type termtype int

var termstr = []string{
	"default", "noempty", "min", "max", "match",
	"idefault", "inoempty", "imin", "imax", "imatch",
}

func (tt termtype) String() string {
	return termstr[int(tt)]
}

type terms []*term

func (tms terms) conflict() {
	tms.conflictDefMinMax("")
	tms.conflictDefMinMax("i")
}

func (tms terms) conflictDefMinMax(iprefix string) {
	var (
		err           error
		def, min, max *term
		at, bt        *term
		name          string
	)
	for _, tm := range tms {
		switch tm.t.String() {
		case iprefix + "default":
			def = tm
		case iprefix + "min":
			min = tm
		case iprefix + "max":
			max = tm
		}
	}

	if def != nil {
		defer func() {
			if v := recover(); v != nil || err != nil {
				panic(fmt.Sprintf("%s: term '%s' and term '%s' are contradictory", name, at, bt))
			}
		}()

		// 这里不管是直接版本还是间接版本统一使用直接版本进行验证.
		if min != nil {
			name, at, bt = min.name, def, min
			if err = template(min.name, "", tmin, min.v, greater)(rft.ValueOf(def.v)); err != nil {
				return
			}
		}
		if max != nil {
			name, at, bt = max.name, def, max
			if err = template(max.name, "", tmax, max.v, less)(rft.ValueOf(def.v)); err != nil {
				return
			}
		}
	}
}

// 重排tms中项的顺序, 当前版本执行的操作有:
// 1. default, idefault会被放在优先处理的位置. 优先
// 处理该项的原因在于保证后续的检测条件是在预设值的
// 基础上进行的. 比如:noempty,default=10. 当未设置
// 该值时应该先进行
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

func (tm term) String() string {
	return fmt.Sprintf("%s=%v", tm.t, tm.v)
}

func newTerm(name, k, v string) *term {
	tm := &term{}
	switch k {
	case "default", "idefault":
		tm.t, tm.v, tm.name = tdefault, getvalue(tdefault, v, name), name
		tm.check = template(name, v, tdefault, tm.v, set)
		if k == "idefault" {
			tm.t, tm.check = tidefault, indirect(name, tidefault, template("", v, tidefault, tm.v, set))
		}
	case "noempty", "inoempty":
		if !isspace(v) {
			panic(fmt.Sprintf("%s: invalid term 'noempty=%s'", name, v))
		}
		tm.t, tm.check, tm.name = tnoempty, noempty(name), name
		if k == "inoempty" {
			tm.t, tm.check = tinoempty, indirect(name, tinoempty, noempty(""))
		}
	case "min", "imin":
		tm.t, tm.v, tm.name = tmin, getvalue(tmin, v, name), name
		tm.check = template(name, v, tmin, tm.v, greater)
		if k == "imin" {
			tm.t, tm.check = timin, indirect(name, timin, template("", v, timin, tm.v, greater))
		}
	case "max", "imax":
		tm.t, tm.v, tm.name = tmax, getvalue(tmax, v, name), name
		tm.check = template(name, v, tmax, tm.v, less)
		if k == "imax" {
			tm.t, tm.check = timax, indirect(name, timax, template("", v, timax, tm.v, less))
		}
	case "match", "imatch":
		if len(v) < 2 || v[0] != '/' || v[len(v)-1] != '/' {
			panic(fmt.Sprintf("%s: invalid term 'match=%s'", name, v))
		}
		tm.t, tm.v, tm.name = tmatch, regexp.MustCompile(v[1:len(v)-1]), name
		tm.check = match(name, tm.v)
		if k == "imatch" {
			tm.t, tm.check = timatch, indirect(name, timatch, match("", tm.v))
		}
	default:
		panic(fmt.Sprintf("%s: unknown term '%s'", name, k))
	}
	return tm
}

func noempty(name string) func(rft.Value) error {
	return func(v rft.Value) error {
		if v.Kind() == rft.Bool {
			panic(fmt.Sprintf("%s: bool type can't support 'noempty' term", name))
		}
		if iszero(v) {
			return fmt.Errorf("%s: is empty", name)
		}
		return nil
	}
}

func match(name string, tv interface{}) func(rft.Value) error {
	return func(v rft.Value) error {
		switch v.Kind() {
		case rft.Array:
			for i := 0; i < v.Len(); i++ {
				if err := match(name, tv)(v.Index(i)); err != nil {
					return err
				}
			}
		case rft.String:
			if re := tv.(*regexp.Regexp); !re.MatchString(v.String()) {
				return fmt.Errorf("%s: '%s' not match '%s'", name, v.String(), re)
			}
		default:
			panic(fmt.Sprintf("%s: %v type can't support 'match' term", name, v.Kind()))
		}
		return nil
	}
}

func template(name string, strv string, tt termtype, tv interface{}, bop func(rft.Value, interface{}) bool) func(rft.Value) error {
	return func(v rft.Value) error {
		// 这个函数主要用于统一bop操作中两个元素的类型.
		var ok bool

		switch v.Kind() {
		case rft.Array:
			for i := 0; i < v.Len(); i++ {
				if err := template(name, strv, tt, tv, bop)(v.Index(i)); err != nil {
					return err
				}
			}
			ok = true
		case rft.Uint, rft.Uint8, rft.Uint16, rft.Uint32, rft.Uint64:
			// 类型转换依赖于termtype类型, 当为(i)min和(i)max的时候
			// 进行类型扩展, 当为(i)default则按照field类型进行转换.
			if tt == tdefault || tt == tidefault {
				switch tv.(type) {
				case uint64:
					// nothing.
				case int64:
					tv = uint64(tv.(int64))
				case time.Duration:
					tv = uint64(tv.(time.Duration))
				case float64:
					tv = uint64(tv.(float64))
				default:
					goto not_match_term
				}

				if v.OverflowUint(tv.(uint64)) {
					goto overflow
				}

			} else {
				// 这里对v重新赋值, 这里并不要求v具备CanSet,
				// 因为比较大小不需要这个属性.
				switch tv.(type) {
				case uint64:
					// nothing.
				case int64, time.Duration:
					v = rft.ValueOf(int64(v.Uint()))
				case float64:
					v = rft.ValueOf(float64(v.Uint()))
				default:
					goto not_match_term
				}
			}
			ok = bop(v, tv)
		case rft.Int, rft.Int8, rft.Int16, rft.Int32, rft.Int64:
			switch tv.(type) {
			case uint64:
				tv = int64(tv.(uint64))
			case int64:
				// nothing.
			case time.Duration:
				tv = int64(tv.(time.Duration))
			case float64:
				if tt == tdefault || tt == tidefault {
					tv = int64(tv.(float64))
				} else {
					v = rft.ValueOf(float64(v.Int()))
				}
			default:
				goto not_match_term
			}

			if (tt == tdefault || tt == tidefault) && v.OverflowInt(tv.(int64)) {
				goto overflow
			}

			ok = bop(v, tv)
		case rft.Float32, rft.Float64:
			switch tv.(type) {
			case uint64:
				tv = float64(tv.(uint64))
			case int64:
				tv = float64(tv.(int64))
			case time.Duration:
				tv = float64(tv.(time.Duration))
			case float64:
				// nothing
			default:
				goto not_match_term
			}

			if (tt == tdefault || tt == tidefault) && v.OverflowFloat(tv.(float64)) {
				goto overflow
			}

			ok = bop(v, tv)
		case rft.Bool, rft.String:
			if tt == tdefault || tt == tidefault {
				if v.Kind() == rft.String {
					ok = bop(v, strv)
				} else {
					ok = bop(v, tv)
				}
				break
			}
			fallthrough
		default:
			panic(fmt.Sprintf("%s: %v type can't support '%s' term", name, v.Kind(), tt))
		}

		if !ok {
			return fmt.Errorf("%s: can't satisfy term '%s=%v'", name, tt, tv)
		}
		return nil

	not_match_term:
		panic(fmt.Sprintf("%s: value of term '%s=%v' not match %s type", name, tt, tv, v.Kind()))
	overflow:
		panic(fmt.Sprintf("%s: value of term '%s=%v' overflow %s", name, tt, tv, v.Kind()))
	}
}

func less(x rft.Value, y interface{}) bool {
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

func greater(x rft.Value, y interface{}) bool {
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

func set(x rft.Value, y interface{}) bool {
	if iszero(x) {
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

func indirect(name string, tt termtype, check func(rft.Value) error) func(rft.Value) error {
	return func(v rft.Value) error {
		// 间接版本只针对Pointer, Interface, Slice, Map.
		switch v.Kind() {
		case rft.Ptr, rft.Interface:
			if !v.IsNil() {
				if err := check(v.Elem()); err != nil {
					return fmt.Errorf("*(%s)%s", name, err)
				}
			}
		case rft.Slice:
			for i := 0; i < v.Len(); i++ {
				if sv := v.Index(i); sv.CanAddr() {
					if err := check(sv); err != nil {
						return fmt.Errorf("%s[%v]%s", name, i, err)
					}
				}
			}
		case rft.Map:
			for _, key := range v.MapKeys() {
				org := v.MapIndex(key)
				sv := rft.New(org.Type()).Elem()
				sv.Set(org)
				if err := check(sv); err != nil {
					return fmt.Errorf("%s[%v]%s", name, key, err)
				}
				v.SetMapIndex(key, sv)
			}
		default:
			panic(fmt.Sprintf("%s: %v type can't support '%s' term", name, v.Kind(), tt))
		}
		return nil
	}
}

func getvalue(tt termtype, v string, name string) interface{} {
	// 标准库https://golang.org/pkg/strconv/#ParseBool
	// 会对0,1进行解释, 这不将这两者看作是bool类型.
	if isbool(v) {
		if tt != tdefault {
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
	} else if tt == tdefault {
		return v
	}

panic_exit:
	panic(fmt.Sprintf("%s: invalid term '%s=%s'", name, tt, v))
	return nil
}

func isspace(s string) bool {
	return regexp.MustCompile(`^[[:space:]]*$`).MatchString(s)
}

// 判断是否为bool值, 这里认为0,1不属于bool类型.
func isbool(s string) bool {
	return regexp.MustCompile(`^(true|True|TRUE|false|False|FALSE)$`).MatchString(s)
}

// 该函数检查0方式为直接引用, 不涉及任何的间接引用.
func iszero(v rft.Value) bool {
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
			z = z && iszero(v.Index(i))
		}
	case rft.Struct:
		for i := 0; i < v.NumField(); i++ {
			z = z && iszero(v.Field(i))
		}
	default:
		// bool, int, uint, float, string
		z = (v.Interface() == rft.Zero(v.Type()).Interface())
	}

	return z
}
