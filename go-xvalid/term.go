// term.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-07
package xvalid

import (
	"fmt"
	rft "reflect"
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
	if re := t.v.(*regexp.Regexp); re.MathString(v.String()) {
		return t.errorf("'%s' not match 'match=%s' term", v.String(), re)
	}
	return nil
}

func (t term) template(v rft.Value, bop func(x, y rft.Value) bool) func(v rft.Value) error {
	return func(v rft.Value) error {
		var err error
		switch v.Kind() {
		case rft.Uint, rft.Uint8, rft.Uint16, rft.Uint32, rft.Uint64:
		case rft.Int, rft.Int8, rft.Int16, rft.Int32, rft.Int64:
		case rft.Float32, rft.Float64:
		case rft.String:
			if t.t == tdefault {
				bop(v, t.v)
				break
			}
			fallthrough
		default:
			t.panic("%v type can't support '%s' term", v.Kind(), t.t)
		}
		return err
	}(v)
}

func (t term) iszero(v rft.Value) bool {
	var (
		z = true
	)

	switch v.Kind() {
	case rft.Map, rft.Slice, rft.Interface, rft.Ptr:
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
		// bool, int, uint, float
		z = (v == rft.Zero(v.Type()))
	}

	return z
}

func (t term) panic(format string, args ...interface{}) {
	panic(fmt.Sprintf("%s: "+format, t.name, args...))
}

func (t term) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("%s: "+format, t.name, args...)
}
