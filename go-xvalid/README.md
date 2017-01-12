#go-xvalid

![Building](https://img.shields.io/badge/building-devel-blue.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)


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

[RE2]: https://golang.org/pkg/regexp/syntax/#hdr-Syntax

