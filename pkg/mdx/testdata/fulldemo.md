
# Markdown Demo

- - -

## 一、标题

### 1. 使用 `#` 表示标题，其中 `#` 号必须在行首，例如：

# 一号标题
## 二号标题
### 三号标题
#### 四号标题
##### 五号标题
###### 六号标题

### 2. 使用 `===` 或者 `---` 表示，例如：

一级标题
===

二级标题
---

#### **扩展：如何换行？**
一般使用 **两个空格** 加 **回车** 换行，不过一些 IDE 也可以直接使用回车换行。


## 二、分割线

使用三个或以上的 `-` 或者 `*`  表示，且这一行只有符号，**注意不要被识别为二级标题即可**，例如中间或者前面可以加空格

- - -

* * *


## 三、斜体和粗体

使用 `*` 和 `**` 分别表示斜体和粗体，例如

*斜体*
**粗体**
***又斜又粗***

#### **扩展：**删除线使用两个 `~` 表示，例如

~~我是要删掉的文字~~

- - -


## 四、超链接和图片

超链接和图片的写法类似，图片仅在超链接前多了一个 `!` ，一般是 [文字描述] (链接)  
两种写法，分别是： [第一种写法](https://www.baidu.com/) 和 [第二种写法][1]  
图片的话就比如这样： ![Image][2]

[1]: https://www.baidu.com/
[2]: https://www.zybuluo.com/static/img/logo.png

- - -


## 五、无序列表

使用 `-`、`+` 和 `*` 表示无序列表，前后留一行空白，可嵌套，例如

+ 一层
    - 二层
    - 二层
        * 三层
            + 四层
+ 一层

- - -


## 六、有序列表

使用 `1. ` （点号后面有个空格）表示有序列表，可嵌套，例如

1. 一层
    1. 二层
    2. 二层
2. 一层

- - -


## 七、文字引用

使用 `>` 表示，可以有多个 `>`，表示层级更深，例如

> 第一层
>>第二层
>这样是跳不出去的
>>> 还可以更深

> 这样就跳出去了

- - -


## 八、行内代码块

其实上面已经用过很多次了，即使用 \` 表示，例如

`行内代码块`

### 扩展：很多字符是需要转义，使用反斜杠 `\` 进行转义

- - -


## 九、代码块

使用四个空格缩进表示代码块，例如

    public class HelloWorld
    {
        public static void main(String[] args)
        { 
            System.out.println( "Hello, World!" );
        }
    }

一些 IDE 支持行数提示和着色，一般使用三个 \` 表示，例如

```
public class HelloWorld
{
    public static void main(String[] args)
    { 
        System.out.println( "Hello, World!" );
    }
}
```

- - -


## 十、表格

直接看例子吧，第二行的 `---:` 表示了对齐方式，默认**左对齐**，还有**右对齐**和**居中**

|商品|数量|单价|
|---|---:|:---:|
|苹果苹果苹果|10|\$1|
|电脑|1|\$1999|

- - -


## 十一、数学公式

使用 `$` 表示，其中一个 \$ 表示在行内，两个 \$ 表示独占一行。
例如质量守恒公式：$$E=mc^2$$
支持 **LaTeX** 编辑显示支持，例如：$\sum_{i=1}^n a_i=0$， 访问 [MathJax][2] 参考更多使用方法。

推荐一个常用的数学公式在线编译网站： [https://www.codecogs.com/latex/eqneditor.php][3]

[2]: http://meta.math.stackexchange.com/questions/5020/mathjax-basic-tutorial-and-quick-reference

[3]: https://www.codecogs.com/latex/eqneditor.php

- - -


## 十二、支持HTML标签

### 1. 例如想要段落的缩进，可以如下：

&nbsp;&nbsp;不断行的空白格&nbsp;或&#160;  
&ensp;&ensp;半方大的空白&ensp;或&#8194;  
&emsp;&emsp;全方大的空白&emsp;或&#8195;


- - -

## 十三、其它
1. markdown 各个 IDE 的使用可能存在大同小异，一般可以参考各个 IDE 的介绍文档
2. 本文档介绍的内容基本适用于大部分的 IDE
3. 其它一些类似 **流程图** 之类的功能，需要看 IDE 是否支持。


查看原始数据：[https://gitee.com/afei_/MarkdownDemo/raw/master/README.md](https://gitee.com/afei_/MarkdownDemo/raw/master/README.md)

博客：[https://blog.csdn.net/afei__/article/details/80717153](https://blog.csdn.net/afei__/article/details/80717153)