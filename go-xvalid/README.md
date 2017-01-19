#go-xvalid

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xvalid**是一个对配置信息合法性进行校验的[Golang][Go]工具包. 它通过对需要检查的字段设定*xvalid tag*.   
提供一系列的限制条件(称为*term*), 来判断配置信息的合法性. 例子如下:

```go

	

	type Person struct {
		Name string `xvalid:"noempty"`
		Age  int `xvalid:"min=1,max=200"`
		Tel  []string `xvalid:"imatch=/^[[:digit:]]{2\,3}-[[:digit:]]+"`
		Friends []*Person `xvalid:"inoempty"`
	}
	
	p := &Person {
		Name: "blinklv",
		Age: 20,
		Tel: []string {
			"086-123456789",
			"079-123456789",
		},
		Friends: []*Person {
			Name: "luna",
			Age: 21,
			Tel: []string{ "086-11111111"},
		},
	}
	if err := xvalid.Validate(&p); err != nil {
		// 处理错误.
	}

```

**Validate**函数的参数必须是*pointer*, *interface*, *map*, *slice*, *array*之一. 一般情况下是指向   
*struct*类型的指针, 如上例所示. 如果*struct*中的字段名为小写或者没有*xvalid tag*, **Validate**则会忽略  
该字段.  
**NOTE**: `,`是**term**的分隔符, 因此当term本身需要包含`,`时应该对其进行转义, 使用`\,`.


## xvalid tag
`xvalid` tag用于配置合法性校验信息, 一个`xvalid` tag由一系列**term**组成.  
**term**是*key=value*形式的键值对, **term**之间使用`,`分割. 现在支持的  
**term**如下:

- **default**: 设定初始值. 支持类型: *bool*, *number*, *string*, *array*, *time.Duration*
- **noempty**: 限定非空. 支持类型: *number*, *string*, *ptr*, *interface*, *struct*, *array*, *slice*, *map*
- **min**: 限定最小值. 支持类型: *number*, *time.Duration*, *array*
- **max**: 限定最大值. 支持类型: *number*, *time.Duration*, *array*
- **match**: 正则匹配. 支持类型: *string*, *array*
- **idefault**: 间接设定初始值. 支持类型: *ptr*, *interface*, *slice*, *map*. 间接支持类型=**default**支持类型.
- **inoempty**: 间接限定非空. 支持类型: *ptr*, *interface*, *slice*, *map*. 间接支持类型=**noempty**支持类型.
- **imin**: 间接限定最小值. 支持类型: *ptr*, *interface*, *slice*, *map*. 间接支持类型=**min**支持类型.
- **imax**: 间接限定最大值. 支持类型: *ptr*, *interface*, *slice*, *map*. 间接支持类型=**max**支持类型.
- **imatch**: 间接正则匹配. 支持类型: *ptr*, *interface*, *slice*, *map*. 间接支持类型=**match**支持类型.

### 直接term
**default**, **noempty**, **min**, **max**, **match**为**直接term**, 相应的使用规则如下:

- **default**
> 如果为大于等于0的整数, 则其类型会被判定为*uint64*, 如果为小于0的整数, 且类型会被判定为*int64*, 其值为  
> 带有小数点的情况则会被判定为*float64*, 但是和**field**的类型发生冲突时会按照**field**的类型进行转换. 你必  
> 须保证对应的类型转是正确的, 同时还要考虑溢出的问题. 合理的转换方向是*uint64*到*int64*, 再到*float64*.  
> 只有在其值为*true*,*True*,*TRUE*,*false*,*False*,*FALSE*中的一个时才会被解释为*bool*类型. 0和1不会被看成*bool*类型.  
> 当以整数开头, `ns`,`us`,`ms`,`s`,`m`,`h`为后缀的情况下会被解释为*time.Duration*类型. 其它情况均被当成字符串. 关于*array*类型的初始值设定稍有不同, 它是将*array*所有子元素设定为同一个值, *array*子元素只能是*default*  
> 支持的类型.

- **noempty**
> 限定该项不可以为空. 对于*number*定义为不能为0, 对于*string*定义为不能空串, 对于*slice*, *map*定义为长度不为0.  
> 对于*array*, *struct*定义为子项至少有一个满足**noempty**的限定, 对于*ptr*, *interface*定义为不能为**nil**.  

- **min**, **max**
> 只有*number*, *time.Duration*均以数值的方式进行比较. *array*类型要求所有子项满足**min**,**max**的限定.

- **match**
> 字符串类型的正则匹配限定. 正则表达式的语法参见[RE2]. *array*类型要求所有子项满足**match**的限定.


### 间接term

间接term是对间接引用的限定, 比如*ptr*和*interface*所指向的值, *slice*和*map*所包含的值. 只有在本身的值合法的情况下  
该项才发挥效果. 比如:

``` go

	type Foo struct {
		A *string `xvalid:"inoempty"`
	}
	
	f1 := &Foo{ A: nil }
	str := ""
	f2 := &Foo{ A: &str }
	
```


`f1`不会被**inoempty**限定, 因为`f1.A`为**nil**, 但是`f2`则会被其限定.  再比如:	

``` go

	type Foo struct {
		A [3]int `xvalid:"idefault=10"`
	}
	
	type Bar struct {
		A *[3]int `xvalid:"idefault=10"`
	}

	f := &Foo{}
	Validate(f)
	
	b := &Bar{}
	Validate(b)

	b.A = &[3]int{}
	Validate(b)
```

`Validate(f)`会导致程序panic, 因为**idefault**只能作用于间接类型. 第一次调用`Validate(b)`没有任何效果,  
第二次调用`Validate(b)`才会设置初始值.

### xvalid的EBNF语法


	  xvalid = space, [ term , [{ "," , term }] ], space
	  term = space, (noempty | min | max | default | match), space;
	  noempty = indirect, "noempty" ;
	  min = indirect, "min", space, "=", expr ;
	  max = indirect, "max", space, "=", expr ;
	  default = indirect, "default", space, "=", expr ;
	  match = indirect, "match", space, "=", space, "/", expr, "/", space;
	  expr = space, [{ character }], space ;
	  indirect = [ "i" ];
	  space = [{ white space }] ;
	  white space = ? white space characters (include CRLF and TAB) ?;
	  character = ? all visible characters ?;
	 
## 关于类型转换
**go-xvalid**在解释**default**, **min**, **max**, **idefault**, **imin**, **imax**这些**term**时, 对其类型的界定顺序是: 

1. *bool* (只针对**default**, **idefault**)
2. *uint64*
3. *int64*
4. *float64*
5. *time.Duration*
6. *string*

只有在值为*true*, *false*, *True*, *False*, *TRUE*, *FALSE*才会被判定为*bool*类型. 如果值大于等于0则判定为*uint64*,  
负整数判定为*int64*, 带有小数点判定为*float64*, 数字开头, 以`ns`,`us`,`ms`,`s`,`m`,`h`为后缀结尾会被判定为*time.Duration*.  
最后一律被判定为*string*. 这样就存在**term**的类型和实际的**field**的类型不匹配的情况. 以下情况是合理的:  

```go

	type Foo struct {
		A uint8 `xvalid:"default=128"`
		B uint16 `xvalid:"max=123456789"`
		C int32 `xvalid:"min=-1234567"`
		D float64 `xvalid:"default=-123.4567"`
		F bool `xvalid:"default=True"`
		G string `xvalid:"default=123456789"`
		H time.Duration `xvalid:"default=20h"`
	}
	
```

注意`G`中的**default**虽然会被判定为*uint64*, 但是在设定`G`的值时又会被转换为*string*, 这是*string*类型特殊之处.

但是如下情况是存在问题的:

```go

	type Foo struct {
		A uint8 `xvalid:"default=1234567"`	   // 溢出
		B int64 `xvalid:"default=40e40"	`      // 40e40超出了int64容纳空间, 因此B的值时未知的
		C int8  `xvalid:"default=string"`	   // 字符串不能赋值个int8
	}


```


[RE2]: https://golang.org/pkg/regexp/syntax/#hdr-Syntax
[Go]: https://golang.org

