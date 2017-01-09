#go-xvalid

![Building](https://img.shields.io/badge/building-devel-blue.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)



## xvalid tag

`xvalid` tag是用于配置合法性校验的信息, 其值由一些term组成.  相应的EBNF语法如下:

      xvalid = term , [{ "," , term }] | "" ;
      term = white space, (noempty | min | max | default | match), white space;
      noempty = "noempty" ;
      min = "min", white space, "=", expr ;
      max = "max", white space, "=", expr ;
      default = "default", white space, "=", expr ;
      match = "match", white space, "=", white space, "/", expr, "/", white space;
      expr = white space, [{ character }], white space ;
      space = [{ white space }] ;
      white space = ? white space characters (include CRLF and TAB) ?;
      character = ? all visible characters ?;

- `noempty`: 表示项不可以空, 对于数值类型就是表示不可以为0.
- `min`: 设置最小值, 只对数值类型有效.
- `max`: 设置最大值, 只对数值类型有效.
- `default`: 设置默认值, 只对数值类型, bool类型, 字符串类型有效.
- `match`: 设置正则匹配, 只对字符串类型有效. 正则语法参见[RE2].





[RE2]: https://github.com/google/re2/wiki/Syntax

