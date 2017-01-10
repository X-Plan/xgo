// xvalid.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-10

// go-xvalid是一个对配置参数进行合法性校验的工具包.
package xvalid

import (
	"flag"
	"fmt"
	rft "reflect"
)

const Version = "1.0.0"

// 并不是所有的类型都可以用xvalid标签修饰, 该数组指出了
// 哪些类型可以支持xvalid tag, 哪些不行.
var support = [...]bool{
	false, true, true, true, true, true, true, true, true,
	true, true, true, false, true, true, false, false, true,
	false, false, true, true, true, true, true, true, false,
}

// 该函数对x进行合法性的校验, x的类型可以为*flag.FlagSet和struct.
// 如果为*flag.FlagSet类型, 则校验规则依赖于Usage中的xvalid标签.
// 如果为一般的struct类型, 则校验规则依赖于'xvalid'tag中的信息.
// xvalid的由一组term构成, term之间用逗号分隔. term的取值如下:
// 1. noempty: 非空. 对于数值类型则为非零.
// 2. min: 最小值. 只对整型, 浮点型和time.Duration类型有效.
// 3. max: 最大值. 只对整型, 浮点型和time.Duration类型有效.
// 4. default: 默认值. 只对标量类型有效.
// 5. match: 正则表达式匹配. 只对字符串类型有效.
//
// 校验过程中会出现两种错误, 第一种是传入的x不符合接口要求, 会直接
// panic, 第二种就是该项的值不符合xvalid的规则, 返回错误.
func Validate(x interface{}) error {
	if fs, ok := x.(*flag.FlagSet); ok {
		return validateFlagSet(fs)
	} else {
		return validate(x)
	}
}

// x的对应的Kind应该为Ptr,Interface,Slice,Array.关于Kind的
// 定义可以参见: https://golang.org/pkg/reflect/#Kind
func validate(x interface{}) (err error) {
	var (
		xv = rft.ValueOf(x)
	)

again:
	switch xv.Kind() {
	case rft.Ptr, rft.Interface:
		if xv = xv.Elem(); xv.Kind() == rft.Struct {
			err = validateStruct(x)
		} else {
			goto again
		}
	case rft.Slice, rft.Array:
		for i := 0; i < xv.Len(); i++ {
			if v := xv.Index(i); v.CanAddr() {
				if err = validate(v.Addr().Interface()); err != nil {
					break
				}
			}
		}
	case rft.Map:
		// Map类型的处理比较特殊, 因为Map中的entity不可寻址.
		// 因此需要先创建一个可寻址的entity, 对其进行完操作
		// 后替换原有的entity.
		for _, key := range xv.MapKeys() {
			org := xv.MapIndex(key)
			v := rft.New(org.Type()).Elem()
			v.Set(org)
			if err = validate(v.Addr().Interface()); err != nil {
				break
			}
			xv.SetMapIndex(key, v)
		}
	}
	return
}

func validateStruct(x interface{}) (err error) {
	var (
		xv    = rft.ValueOf(x).Elem()
		xt, n = xv.Type(), xv.NumField()
	)

	for i := 0; i < n; i++ {
		var (
			fv, sf     = xv.Field(i), xt.Field(i)
			kind, name = fv.Kind(), sf.Name
		)

		if tag, ok := sf.Tag.Lookup("xvalid"); ok && fv.CanSet() {
			// 即便是xvalid tag的值为空, 但是只要设置了该tag.
			// 当类型不匹配的时候依然会导致程序panic.
			if !support[int(kind)] {
				panic(fmt.Sprintf("%s: %v type can't support 'xvalid' tag", name, kind))
			}

			tms := newTerms(name, tag)
			tms.conflict()
			tms.resort()

			for _, tm := range tms {
				if err = tm.check(fv); err != nil {
					return
				}
			}

			switch kind {
			case rft.Struct:
				err = validateStruct(fv.Addr().Interface())
			case rft.Ptr, rft.Interface, rft.Slice, rft.Array, rft.Map:
				err = validate(fv.Interface())
			}
		}
	}

	return
}

func validateFlagSet(fs *flag.FlagSet) error {
	return nil
}