# XML DOM Go库
## 项目简介
`dom4g` 是基于 Go 标准库 `encoding/xml` 封装的完整DOM操作库，对标Java dom4j设计理念，提供完整XML解析、命名空间支持、节点增删改查、XPath基础查询、并发安全、格式化序列化、CDATA特殊处理等能力。
### 核心特性
1. **完整DOM节点模型**：统一`Node`顶层接口，支持Element/Text/CDATA/Comment/ProcInst五类标准XML节点
2. **并发安全树模型**：整树读写锁+原子计数，支持多协程并发读写/修改无数据竞争
3. **丰富查询API**：路径查询`GetNodeByPath`、递归标签检索、基础XPath子集（//递归/索引/属性过滤）
4. **完善节点操作**：增/删/插/替换/深克隆、属性CRUD、文本/CDATA/注释单独操作
5. **安全防护**：默认开启XXE外部实体拦截、最大解析深度防栈溢出、空白文本自动裁剪
6. **格式化输出**：压缩单行输出 + 带缩进格式化XML，支持输出完整文档/节点OuterXML/InnerXML

# 使用教程

## 1. 安装引入
### 模块导入
```go
import "github.com/donnie4w/dom4g"
```
### 依赖
仅依赖Go标准库，无第三方依赖

## 2. 快速入门
### 基础加载与遍历
```go
package main

import (
	"fmt"
	"github.com/donnie4w/dom4g"
)

func main() {
	xmlStr := `<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="demo">
	<book id="001">Go编程</book>
</root>`
	// 加载XML
	root, err := dom4g.LoadByXml(xmlStr)
	if err != nil {
		panic(err)
	}
	// 命名空间查询
	uri := root.LookupNamespaceURI("xsi")
	fmt.Println("xsi命名空间URI：", uri)
	// 节点查询
	book := root.Child("book")
	fmt.Println("图书文本：", book.Text())
	// 属性读取
	id, ok := book.AttrValue("id")
	fmt.Println("id=", id, ok)
	// 格式化输出
	fmt.Println(root.ToXMLIndent("", "  "))
}
```

### 动态构造XML
```go
func buildXML() {
	root := dom.NewElement("root")
	// 添加命名空间属性
	root.AddAttr("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance")
	root.AddAttr("xsi:type", "demo")
	// 新增子节点
	book := dom.NewElement("book").AddAttr("id", "001").AddText("Go编程")
	_ = root.AddChild(book)
	// 输出完整文档
	fmt.Println(root.ToXML())
}
```

## 3. 核心API文档
### 3.1 XML加载函数
| 函数 | 说明 |
|------|------|
| `LoadByXml(xmlstr string) (*Element, error)` | 默认配置加载字符串XML |
| `LoadByXmlOption(xmlstr string, opt ParseOption)` | 自定义解析配置加载 |
| `LoadByStream(r io.Reader)` | 从IO流加载 |
| `LoadByStreamOption(r io.Reader, opt ParseOption)` | 流+自定义配置 |
| `LoadByXmlExtend(xmlstr)` | CDATA增强解析，解决标准库CDATA拆分丢失 |
| `LoadByStreamExtend(r io.Reader)` | 流CDATA增强解析 |

**ParseOption 配置项**
```go
type ParseOption struct {
	MaxDepth        int  // 最大解析深度，默认1000，防止递归栈溢出
	TrimWhitespace  bool // 是否裁剪纯空白Text节点，默认true
	DisableXXE      bool // 禁用XXE外部实体，默认开启
	PreserveComment bool // 是否保留注释节点，默认true
}
```

### 3.2 元素创建
```go
// 普通元素（默认非并发安全）
NewElement(elementName string) *Element
// 直接创建开启并发安全的根元素
NewSyncElement(elementName string) *Element
```

### 3.3 命名空间核心方法
```go
// 根据前缀查找对应命名空间URI
LookupNamespaceURI(prefix string) string
// 根据URI反向查找前缀
LookupPrefix(uri string) string
```

### 3.4 节点查询API
#### 基础查询
```go
Child(name string) *Element              // 获取第一个同名直接子元素
Children(name string) []*Element        // 获取所有同名直接子元素
AllChildren() []*Element                // 获取全部直接子元素（过滤文本等节点）
FirstChild() *Element                   // 首个子元素
LastChild() *Element                    // 最后一个子元素
GetElementsByTagName(name string) []*Element // 递归全树匹配标签
```
#### 路径查询
```go
GetNodeByPath(path string) *Element     // 单节点路径查询，支持相对路径
GetNodesByPath(path string) []*Element  // 批量路径匹配
```
#### XPath查询（基础子集）
```go
Find(xpath string) (*Element, error)    // 获取首个匹配节点
FindAll(xpath string) ([]*Element, error) // 获取全部匹配
支持语法：
1. 绝对/相对路径：root/book/title
2. 递归匹配 //book
3. 索引过滤 book[1]
4. 属性过滤 //book[@id="1001"]
```

#### 遍历
```go
Walk(fn func(el *Element) bool)        // 快照模式遍历，回调内可安全修改，无死锁
WalkUnsafe(fn func(el *Element) bool)  // 持锁只读遍历，高性能，禁止写操作
```

### 3.5 节点增删改插
```go
AddChild(el *Element) error                     // 追加子元素
AddChildByString(xmlstr string) error           // 从XML字符串解析并追加
InsertBefore(new, ref *Element) error           // 在指定节点前插入
InsertAfter(new, ref *Element) error             // 在指定节点后插入
ReplaceChild(old, new *Element) error            // 替换子节点
RemoveChild(el *Element) bool                   // 删除指定节点实例
RemoveNodeByName(name string) bool              // 删除所有同名直接子元素
RemoveAll()                                     // 清空全部子节点
Clone(deep bool) Node                           // 节点克隆（deep=true深拷贝）
CloneElement(deep bool) *Element                // 强转元素深拷贝
```

### 3.6 属性操作
```go
AttrValue(name string) (val string, exist bool) // 获取属性
HasAttr(name string) bool                       // 判断属性存在
AddAttr(name, val string)                      // 新增/覆盖属性
RemoveAttr(name string) bool                    // 删除属性
```

### 3.7 文本/CDATA/注释/PI
```go
Text() string       // 当前节点下的纯文本
InnerText() string          // 合并所有后代纯文本
AddText(content string) *Element   // 追加文本节点
SetText(content) *Element         // 清空原有文本并设置
ClearTexts() *Element             // 清空所有直接文本节点

AddCDATA(raw string) *Element     // 添加CDATA段
SetCDATA(raw) *Element            // 重置CDATA
ClearCDATAs() *Element            // 清空CDATA
CDATAs() []string                 // 获取所有后代CDATA内容

AddComment(text string) *Element  // 添加注释
AddProc(target, data string)      // 添加处理指令<?xxx ...?>
```

### 3.8 序列化输出
```go
ToString() string    // 当前节点OuterXML（自身+子节点）
InnerXML() string    // 仅子节点XML，不含自身标签
ToXML() string       // 输出完整文档（带xml头）
ToXMLIndent(prefix, indent string) // 格式化缩进输出
SyncToXml() string   // 兼容并发输出别名
Head() string        // 获取XML声明头 <?xml ...?>
```

### 3.9 并发安全控制
仅根节点生效，整树共享读写锁
```go
EnableSync()  // 开启并发读写保护
DisableSync() // 关闭锁，最大化单线程性能
```
- 读操作：RLock() 共享读
- 写操作：Lock() 独占写
- Walk快照模式：先拷贝节点列表再遍历，回调修改无死锁

### 3.10 节点导航与统计
```go
// 树导航
Parent() *Element
Root() *Element
PreviousSibling() Node
NextSibling() Node
Ancestors() []*Element  // 所有祖先节点
Siblings() []*Element   // 全部兄弟节点
Depth() int             // 节点深度（根=1）
NodePath() string       // 完整节点路径 root/a/b/c

// 状态判断
IsRoot() bool   // 是否根
IsLeaf() bool   // 是否叶子
HasChildNodes() bool //是否有子节点

// O(1)原子计数（无需遍历）
ChildrenLength() int64          // 直接子元素数量
ChildLengthByName(name) int64   // 同名直接子元素数量
DocLength() int64               // 当前子树全部元素总数（含自身）
```

## 4. 并发安全机制
1. **单树全局读写锁sync.RWMutex**：所有节点共享根锁，减少内存占用，避免每个节点独立锁的性能损耗
2. **原子标记isSync**：无锁场景自动跳过加解锁，单线程无额外开销
3. **原子计数childCount/descCount**：增删节点时原子更新，读取无需加锁，O(1)统计
4. **两种遍历模式**
   - `Walk`：快照遍历，先拷贝节点列表，遍历过程可增删改节点，无死锁（推荐混合读写场景）
   - `WalkUnsafe`：持锁原地遍历，只读高性能，回调禁止写操作（死锁风险）
5. 所有修改API（Add/Remove/Insert/Replace/AddAttr）内部自动获取写锁，查询API自动读锁，用户无需手动管理锁

## 5. XPath 支持范围
### 支持语法
1. 层级路径：`root/book/title`
2. 全局递归匹配：`//book`
3. 位置索引：`//book[2]`（从1开始）
4. 属性等值过滤：`//book[@id="1001"]`
### 暂不支持
函数、通配符`*`、或`|`、多条件、文本匹配、轴运算（后续迭代扩展）

## 6. 内置安全策略
1. **XXE防护**：默认`DisableXXE=true`，覆盖charsetReader拦截外部实体读取，忽略DTD指令
2. **最大解析深度**：默认1000层，超深度直接返回`ErrMaxDepthExceeded`，防止恶意嵌套XML栈溢出
3. 空白文本裁剪：默认开启，减少无用Text节点内存占用
4. 解析panic捕获：解析崩溃自动包装`ParseError`带堆栈信息，不会导致服务宕机


## 7. 测试与基准测试
### 单元测试覆盖  见 `dom_test.go`
1. XML加载、基础CRUD
2. 命名空间解析与反向查找
3. CDATA/注释/处理指令处理
4. XPath路径查询、标签递归查找
5. 并发读写、混合增删压力测试
6. 深克隆隔离验证
7. 格式化输出、特殊字符转义
8. 解析深度、XXE安全校验
9. Walk两种遍历死锁验证

### Bench基准测试  见 `dom_test.go`
内置基准函数：
- 小/大XML加载性能
- ToXML序列化单行/带锁序列化
- 节点查询、路径查询
- 全树遍历
- 新增节点、新增属性
- 深克隆性能


## 8. 已知局限
### 当前局限
1. XPath仅支持基础子集，不支持函数、多条件、通配符
2. 不支持XML Schema/DTD校验


## 9. 完整示例（命名空间+并发+XPath）
```go
package main

import (
	"fmt"
	"sync"
	"github.com/donnie4w/dom4g"
)

func main() {
	xmlSrc := `<?xml version="1.0" encoding="utf-8"?>
<library xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<book xsi:type="BookType" id="1001">Go高性能开发</book>
	<book xsi:type="BookType" id="1002">Goroutine并发实战</book>
</library>`
	// 加载
	root, err := dom4g.LoadByXml(xmlSrc)
	if err != nil {
		panic(err)
	}
	// 命名空间查询
	fmt.Println("xsi URI:", root.LookupNamespaceURI("xsi"))
	// XPath 查询id=1002的书籍
	target, _ := root.Find(`//book[@id="1002"]`)
	fmt.Println("匹配书籍：", target.Text())
	// 并发读写
	root.EnableSync()
	var wg sync.WaitGroup
	wg.Add(2)
	// 读协程
	go func() {
		defer wg.Done()
		fmt.Println(root.ToXMLIndent("", "  "))
	}()
	// 写协程
	go func() {
		defer wg.Done()
		newBook := dom4g.NewElement("book")
		newBook.AddAttr("id", "1003")
		newBook.AddAttr("xsi:type", "BookType")
		newBook.AddText("XML DOM实战")
		_ = root.AddChild(newBook)
	}()
	wg.Wait()
	root.DisableSync()
	// 深克隆
	cloneRoot := root.Clone(true).(*dom4g.Element)
	fmt.Println("克隆后文档：\n", cloneRoot.ToXMLIndent("", "  "))
}
```

## 10. 错误常量说明
```go
ErrNilNode          // 节点为nil
ErrNotFound         // 引用节点不存在
ErrInvalidName      // 非法元素名
ErrParse            // 通用解析错误
ErrMaxDepthExceeded // 超出最大解析深度
ErrXXEBlocked       // 拦截外部实体
ErrInvalidXPath     // XPath语法错误
ErrUnsupportedXPath // XPath不支持语法
```
`ParseError` 携带Line（字节偏移），可获取解析出错位置与原始错误。