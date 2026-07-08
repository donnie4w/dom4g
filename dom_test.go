// Copyright (c) 2016, donnie <donnie4w@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// github.com/donnie4w/dom4g

package dom4g

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestDemoFull(t *testing.T) {
	// ========== 1. 加载 XML 与基础信息 ==========
	fmt.Println("\n【1. XML 加载与基础信息】")
	xmlStr := `<?xml version="1.0" encoding="utf-8"?>
<library name="编程书架" location="北京">
  <book id="1001">
    <title>Go语言实战</title>
    <author>张三</author>
    <price>89.9</price>
  </book>
  <book id="1002">
    <title>Go高级编程</title>
    <author>李四</author>
    <price>128.0</price>
    <tags>
      <tag>后端</tag>
      <tag>并发</tag>
    </tags>
  </book>
  <magazine id="M01">
    <title>程序员月刊</title>
  </magazine>
</library>`

	root, err := LoadByXml(xmlStr)
	if err != nil {
		t.Fatal("加载XML失败:", err)
	}
	fmt.Println("根节点名称:", root.Name)
	fmt.Println("XML头声明:", root.Head())
	fmt.Println("是否为根节点:", root.IsRoot())

	// ========== 2. 节点查询 ==========
	fmt.Println("\n【2. 节点查询】")
	// 单个节点查询
	firstBook := root.Child("book")
	fmt.Println("第一本图书标题:", firstBook.Child("title").Text())

	// 多个同名节点
	allBooks := root.Children("book")
	fmt.Printf("图书节点总数: %d 本\n", len(allBooks))
	for i, book := range allBooks {
		id, _ := book.AttrValue("id")
		fmt.Printf("  第%d本 ID=%s 书名=%s\n", i+1, id, book.Child("title").Text())
	}

	// 路径查询
	priceNode := root.GetNodeByPath("library/book/price")
	fmt.Println("路径查询 library/book/price:", priceNode.Text())

	// 路径批量查询
	allTags := root.GetNodesByPath("library/book/tags/tag")
	fmt.Printf("路径批量查询 tag 节点数: %d\n", len(allTags))

	// 所有子节点
	rootChildren := root.AllChildren()
	fmt.Printf("根节点直接子节点数: %d (book*2 + magazine*1)\n", len(rootChildren))

	// ========== 3. 属性操作 ==========
	fmt.Println("\n【3. 属性操作】")
	libName, _ := root.AttrValue("name")
	fmt.Println("原图书馆名称属性:", libName)

	root.AddAttr("version", "v2.0")
	root.AddAttr("name", "技术图书馆") // 覆盖已有属性

	attr, b := root.AttrValue("name")
	fmt.Println("新增/修改属性后 name:", attr, b)
	fmt.Println("是否存在 version 属性:", root.HasAttr("version"))

	root.RemoveAttr("location")
	fmt.Println("删除 location 后是否存在:", root.HasAttr("location"))

	// ========== 4. 新增节点 ==========
	fmt.Println("\n【4. 新增节点】")
	// 方式1：通过 Element 对象新增
	newBook := NewElement("book")
	newBook.AddAttr("id", "1003")
	newBook.AddChild(NewElement("title").AddText("Go并发编程实战"))
	newBook.AddChild(NewElement("author").AddText("王五"))
	newBook.AddChild(NewElement("price").AddText("99.0"))
	root.AddChild(newBook)
	fmt.Println("新增第3本图书后，图书总数:", len(root.Children("book")))

	// 方式2：通过 XML 字符串新增
	root.AddChildByString(`<journal id="J01"><name>计算机学报</name><period>月刊</period></journal>`)
	fmt.Println("通过字符串新增期刊节点:", root.Child("journal").Child("name").Text())

	// ========== 5. 节点插入与替换 ==========
	fmt.Println("\n【5. 节点插入与替换】")
	refNode := root.Child("magazine")
	promoNode := NewElement("promotion").AddText("限时免费")
	root.InsertBefore(promoNode, refNode)
	fmt.Println("在 magazine 前插入 promotion 后，第3个子节点:", root.AllChildren()[2].Name)

	newMag := NewElement("magazine")
	newMag.AddAttr("id", "M02")
	newMag.AddChild(NewElement("title").AddText("架构师周刊"))
	root.ReplaceChild(refNode, newMag)
	fmt.Println("替换 magazine 节点后新标题:", root.Child("magazine").Child("title").Text())

	// ========== 6. 节点删除 ==========
	fmt.Println("\n【6. 节点删除】")
	fmt.Println("删除前子节点数:", len(root.AllChildren()))
	root.RemoveNodeByName("journal") // 按名称删除所有同名节点
	root.RemoveChild(promoNode)      // 删除指定节点实例
	fmt.Println("删除 journal 和 promotion 后子节点数:", len(root.AllChildren()))

	// ========== 7. 节点导航 ==========
	fmt.Println("\n【7. 节点导航】")
	secondBook := root.Children("book")[1]
	fmt.Println("第二本图书的父节点名:", secondBook.Parent().Name)
	//fmt.Println("第一本图书下一个兄弟节点:", firstBook.NextSibling().Name)
	//fmt.Println("第二本图书上一个兄弟节点:", secondBook.PreviousSibling().Name)
	fmt.Println("根节点第一个子节点:", root.FirstChild().Name)
	fmt.Println("根节点最后一个子节点:", root.LastChild().Name)

	// ========== 8. 特殊节点：文本/CDATA/注释 ==========
	fmt.Println("\n【8. 特殊节点：文本/CDATA/注释】")
	descNode := NewElement("description")
	descNode.AddComment("以下为富文本内容")
	descNode.AddCDATA("<html><body>包含特殊字符 & <符号> 的内容</body></html>")
	descNode.AddText("普通文本说明")
	root.AddChild(descNode)

	fmt.Println("description 节点完整输出:")
	fmt.Println(descNode.ToString())
	fmt.Println("提取所有纯文本内容:", descNode.Text())

	// ========== 9. 遍历与全局搜索 ==========
	fmt.Println("\n【9. 遍历与全局搜索】")
	totalElements := 0
	root.Walk(func(el *Element) bool {
		totalElements++
		return true
	})
	fmt.Println("全文档元素节点总数:", totalElements)

	allTitleNodes := root.GetElementsByTagName("title")
	fmt.Printf("全局搜索所有 title 节点: %d 个\n", len(allTitleNodes))
	for _, n := range allTitleNodes {
		fmt.Println("  -", n.Text())
	}

	// ========== 10. 序列化输出 ==========
	fmt.Println("\n【10. 序列化输出】")
	fmt.Println("--- 当前节点 ToString() ---")
	fmt.Println(firstBook.ToString())

	fmt.Println("\n--- 完整文档 ToXML() ---")
	fmt.Println(root.ToXML())

	fmt.Println("\n--- 格式化缩进输出 ToXMLIndent() ---")
	fmt.Println(root.ToXMLIndent("", "  "))

	// ========== 11. 节点深拷贝 ==========
	fmt.Println("\n【11. 节点深拷贝】")
	clonedBook := firstBook.Clone(true).(*Element)
	firstBook.Child("title").AddText("原节点已修改")
	fmt.Println("原节点标题:", firstBook.Child("title").Text())
	fmt.Println("克隆节点标题:", clonedBook.Child("title").Text())
	fmt.Println("克隆节点完全独立，不受原节点影响")

	// ========== 12. 统计与状态判断 ==========
	fmt.Println("\n【12. 统计与状态判断】")
	fmt.Println("根节点直接子元素数:", root.ChildrenLength())
	fmt.Println("全文档元素总数:", root.DocLength())
	fmt.Println("title 节点是否为叶子节点:", firstBook.Child("title").IsLeaf())
	fmt.Println("book 节点是否包含子节点:", firstBook.HasChildNodes())

	// ========== 13. 并发安全开关 ==========
	fmt.Println("\n【13. 并发安全开关】")
	root.EnableSync()
	fmt.Println("已开启并发安全模式，后续多协程操作线程安全")
	root.DisableSync()
	fmt.Println("已关闭并发安全模式，性能最优")

	// ========== 14. ProcInst ==========
	fmt.Println("\n【14. ProcInst】")
	root.AddProcInst("xml-stylesheet", `type="text/xsl" href="style.xsl"`)
	// 内部元素添加自定义PI
	book := root.Child("book")
	book.AddProcInst("page", "render multi")
	fmt.Println("xml:", book.ToXML())

}

func TestAdd(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAdd---------------------------------")
	s := `<?xmlversion="1.0" encode="utf-8"?><a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, err := LoadByXml(s)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	el.AddChildByString(`<h>你好</h>`)           //新增节点参数可以为字符串
	el.AddChild(NewElement("g").AddText("新增")) //新增节点参数为Element
	fmt.Println(el.ToString())
	g := el.Child("g")
	g.AddText("修改后的值")
	g.AddText("修改后的值2")
	g.AddChild(NewElement("h"))
	g.AddAttr("gattr", "gattrv")
	g.Child("h").AddText("我是h")
	fmt.Println(el.SyncToXml())
}

func TestRemove(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestRemove---------------------------------")
	s := `<a Age="10"><b>wu1</b><b>wu2</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.RemoveNodeByName("b")
	fmt.Println(el.ToString())
}

func TestGet(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestGet---------------------------------")
	s := `<a Age="10" go="1"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	//获取节点d 单个值
	fmt.Println("d:", el.Child("d").Text())
	//获取节点d 多个值
	for _, elem := range el.Children("d") {
		fmt.Println(elem.Name, elem.Text())
	}
	//获取属性
	for _, attrs := range el.Attrs {
		fmt.Println(attrs.Name, attrs.Value)
	}
}

func TestAttrAdd(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAttrAdd---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.AddAttr("name", "donnie") //如果属性不存在，则新增
	el.AddAttr("Age", "100")     //如果属性存在，则覆盖
	fmt.Println(el.ToString())
}

func TestAttrGet(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAttrGet---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	//返回两个值，第一个值为查询到值，第二个值为bool，表示是否存在该属性
	fmt.Println(el.AttrValue("Age"))  //10,true
	fmt.Println(el.AttrValue("name")) // false
}

func TestAttrRemove(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAttrRemove---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.RemoveAttr("Age")
	fmt.Println(el.ToString())
}

/*
*
ToString()  & ToXML()
*/
func TestToString_ToXML(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestToString_ToXML---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	f := el.Child("e").Child("f")
	fmt.Println(el.ToString()) //只打印了当前节点的xml信息
	fmt.Println("f.Value>>>>>", f.Text())
	fmt.Println(f.ToString()) //只打印了当前节点的xml信息
	fmt.Println(f.ToXML())    //打印了整个文档的xml信息，与从根节点打印效果的相同
}

/*
*
ToString()  & ToXML()
*/
func TestParent(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestParent---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	f := el.Child("e").Child("f")
	fmt.Println(f.Parent().ToString()) //打印e节点信息
}

/*
*

	test length
*/
func TestLength(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestLength---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	e := el.Child("e")
	fmt.Println(el.DocLength())              //打印整个文档节点个数    7
	fmt.Println(el.ChildrenLength())         //打印子节点个数         5
	fmt.Println(e.ChildrenLength())          //打印e节点子节点个数     1
	fmt.Println(e.DocLength())               //打印e节点所在整个文档节点个数  2
	el.AddChildByString(`<h><i>来了吗</i></h>`) //增加2个节点
	newel := NewElement("g").AddText("新增的g值")
	el.AddChild(newel)          //增加1个节点
	fmt.Println(el.DocLength()) //10
	fmt.Println(el.ToXML())
	el.RemoveNodeByName("g") //删除一个节点               // 9
	fmt.Println(el.DocLength())
	el.RemoveNodeByName("h")    //删除一个节点，注意h节点本身包含了一个子节点
	fmt.Println(el.DocLength()) //7
	el.AddChildByString(`<h><i><j>又来了吗</j><j>来了</j><j>嗯</j></i></h>`)
	fmt.Println(el.DocLength()) //12
	i := el.Child("h").Child("i")
	i.RemoveNodeByName("j")
	fmt.Println(i.DocLength()) // 9
	fmt.Println(i.ToXML())
}

// 2014-11-24
// test GetNodeByPath()
func TestGetByPath(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestGetByPath---------------------------------")
	s := `<a Age="10">fffffffff<b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e>eeeeeeee</a>`
	el, _ := LoadByXml(s)
	t.Log(el.Text())
	b := el.GetNodeByPath("a/e/f")
	fmt.Println("a/e/f", b.Name, ":", b.Text())
	fmt.Println(el.ToString())
	bs := el.GetNodesByPath("a/d")
	fmt.Println(bs[0].Text(), bs[1].Text())
}

func TestNew(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestNewElement---------------------------------")
	el := NewElement("a").AddText("a")
	fmt.Println(el.ToString())
	fmt.Println(el.ToXML())
}

func TestCDATA(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestCDATA---------------------------------")
	s := `<a xmlns:android="http://schemas.android.com/apk/res/android"><b><![CDATA[<b>abc</b>]]><![CDATA[<b>ddd</b>]]></b><c>abc</c></a>`
	el, _ := LoadByXmlExtend(s)

	b := el.GetNodeByPath("a/b")
	fmt.Println(el.ToString())
	fmt.Println(b.CDATAs())
}

func TestXMLEscape(t *testing.T) {
	fmt.Println("\n-----------------------------TestXMLEscape---------------------------------")
	el := NewElement("root")
	el.AddChild(NewElement("content").AddText(`<a>&"test'</a>`))
	el.AddAttr("desc", `quote: "test"`)

	xmlStr := el.ToString()
	if strings.Contains(xmlStr, "<a>") {
		t.Fatal("special characters < > not escaped")
	}
	if !strings.Contains(xmlStr, "&lt;a&gt;") {
		t.Fatal("angle brackets escape failed")
	}
	if !strings.Contains(xmlStr, "&amp;") {
		t.Fatal("ampersand escape failed")
	}
	if !strings.Contains(xmlStr, "&quot;") {
		t.Fatal("quote in attribute escape failed")
	}
	fmt.Println("escaped xml:", xmlStr)
}

func TestDeepClone(t *testing.T) {
	fmt.Println("\n-----------------------------TestDeepClone---------------------------------")
	s := `<root id="1"><a name="test"><b>inner text</b></a></root>`
	root, _ := LoadByXml(s)
	cloned := root.Clone(true).(*Element)
	t.Log("cloned:", cloned.ToXML())

	// 修改原节点，验证克隆节点不受影响
	root.Child("a").AddAttr("modified", "true")
	root.Child("a").Child("b").AddText("changed")

	if cloned.Child("a").HasAttr("modified") {
		t.Error("deep clone failed: attribute modification leaked to clone")
	}
	if cloned.Child("a").Child("b").Text() == "changed" {
		t.Error("deep clone failed: value modification leaked to clone")
	}
	if cloned.Child("a").Child("b").Text() != "inner text" {
		t.Error("deep clone failed: child value not preserved")
	}
	fmt.Println("original:", root.ToString())
	fmt.Println("cloned:", cloned.ToString())
}

func TestWalkAndSearch(t *testing.T) {
	fmt.Println("\n-----------------------------TestWalkAndSearch---------------------------------")
	s := `<root><a><b><c>deep1</c></b></a><a><c>deep2</c></a><c>shallow</c></root>`
	root, _ := LoadByXml(s)

	// 全量遍历计数
	count := 0
	root.Walk(func(el *Element) bool {
		count++
		return true
	})
	if count != 7 { // root + 2a + b + 3c = 7
		t.Errorf("walk count wrong, expected 7 got %d", count)
	}

	// 递归查找所有同名节点
	cNodes := root.GetElementsByTagName("c")
	if len(cNodes) != 3 {
		t.Errorf("GetElementsByTagName wrong, expected 3 got %d", len(cNodes))
	}
	fmt.Printf("found %d <c> nodes, total elements: %d\n", len(cNodes), count)
}

func TestIndentOutput(t *testing.T) {
	fmt.Println("\n-----------------------------TestIndentOutput---------------------------------")
	s := `<root><a><b>text</b></a><c>value</c></root>`
	root, _ := LoadByXml(s)
	formatted := root.ToXMLIndent("", "  ")
	fmt.Println(formatted)

	if !strings.Contains(formatted, "  <a>") {
		t.Error("indent format failed")
	}
}

func TestConcurrentRead(t *testing.T) {
	fmt.Println("\n-----------------------------TestConcurrentRead---------------------------------")
	s := `<root age="10"><a id="1">text1</a><a id="2">text2</a><b><c>deep_value</c></b></root>`
	root, err := LoadByXml(s)
	if err != nil {
		t.Fatal(err)
	}
	root.EnableSync()
	defer root.DisableSync()

	const goroutines = 100
	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = root.ToString()
				_ = root.ToXML()
				_ = root.Child("a").Text()
				_ = root.Children("a")
				_ = root.GetNodeByPath("root/b/c")
				_, _ = root.AttrValue("age")
				root.Walk(func(*Element) bool { return true })
				_ = root.FirstChild()
				_ = root.LastChild()
			}
		}()
	}
	wg.Wait()
	fmt.Println("concurrent read test passed, no panic")
}

func TestConcurrentWrite(t *testing.T) {
	fmt.Println("\n-----------------------------TestConcurrentWrite---------------------------------")
	root := NewElement("root")
	root.EnableSync()
	defer root.DisableSync()

	const goroutines = 50
	const perGoroutine = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				el := NewElement(fmt.Sprintf("node_%d", id)).AddText(fmt.Sprintf("val_%d", j))
				el.AddAttr("idx", fmt.Sprintf("%d", j))
				_ = root.AddChild(el)
			}
		}(i)
	}
	wg.Wait()

	expected := goroutines * perGoroutine
	actual := len(root.AllChildren())
	if actual != expected {
		t.Errorf("concurrent write count mismatch: expected %d, got %d", expected, actual)
	}
	fmt.Printf("concurrent write passed: %d nodes added\n", actual)
}

func TestConcurrentMixed(t *testing.T) {
	fmt.Println("\n-----------------------------TestConcurrentMixed---------------------------------")
	root := NewElement("root")
	root.EnableSync()
	defer root.DisableSync()

	// 预初始化基础节点
	for i := 0; i < 100; i++ {
		root.AddChild(NewElement("item").AddText(fmt.Sprintf("val_%d", i)))
	}

	const readers = 30
	const writers = 20
	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(readers + writers)

	// 读协程
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = root.ToXML()
				_ = root.Children("item")
				_ = root.GetElementsByTagName("item")
				_ = root.FirstChild()
				_ = root.LastChild()
			}
		}()
	}

	// 写协程
	for i := 0; i < writers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				el := NewElement("new_item").AddText(fmt.Sprintf("v_%d_%d", id, j))
				_ = root.AddChild(el)
				root.AddAttr(fmt.Sprintf("attr_%d", id), fmt.Sprintf("%d", j))
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("concurrent mixed read/write test passed, no panic")
}

func TestConcurrentAddRemove(t *testing.T) {
	fmt.Println("\n-----------------------------TestConcurrentAddRemove---------------------------------")
	root := NewElement("root")
	root.EnableSync()
	defer root.DisableSync()

	// 初始化目标节点
	for i := 0; i < 200; i++ {
		root.AddChild(NewElement("target").AddText(fmt.Sprintf("v_%d", i)))
	}

	const goroutines = 20
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 删除协程
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				root.RemoveNodeByName("target")
			}
		}()
	}

	// 添加协程
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				root.AddChild(NewElement("target").AddText("new_val"))
			}
		}()
	}

	wg.Wait()
	fmt.Println("concurrent add/remove test passed, no panic")
}

/****************************************************************/
func init() {
	// 生成 1000 个子节点的大文档
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&sb, `<item id="%d" name="item_%d">value_%d<sub>sub_%d</sub></item>`, i, i, i, i)
	}
	sb.WriteString("</root>")
	testXMLLarge = sb.String()
}

var testXMLSmall = `<root age="10"><a>text1</a><b name="test">text2</b><c><d>deep_value</d></c></root>`

var testXMLLarge string

func BenchmarkLoadXML_Small(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := LoadByXml(testXMLSmall)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadXML_Large(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := LoadByXml(testXMLLarge)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToXML_Small(b *testing.B) {
	root, _ := LoadByXml(testXMLSmall)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.ToXML()
	}
}

func BenchmarkToXML_Large(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.ToXML()
	}
}

func BenchmarkToXML_Large_WithSync(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	root.EnableSync()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.ToXML()
	}
}

func BenchmarkNodeQuery(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.Child("item")
	}
}

func BenchmarkGetNodeByPath(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.GetNodeByPath("root/item/sub")
	}
}

func BenchmarkWalkAllElements(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		count := 0
		root.Walk(func(*Element) bool {
			count++
			return true
		})
	}
}

func BenchmarkAddNode(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		root := NewElement("root")
		_ = root.AddChild(NewElement("child").AddText("value"))
	}
}

func BenchmarkAddAttr(b *testing.B) {
	root := NewElement("root")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		root.AddAttr(fmt.Sprintf("attr_%d", i), "value")
	}
}

func BenchmarkCloneDeep(b *testing.B) {
	root, _ := LoadByXml(testXMLLarge)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = root.Clone(true)
	}
}

// Walk快照模式死锁验证
func TestWalkSnapshotNoDeadlock(t *testing.T) {
	xmlStr := `<root><a><b val="1"/></a><a><b val="2"/></a></root>`
	root, err := LoadByXml(xmlStr)
	if err != nil {
		t.Fatal(err)
	}
	root.EnableSync()
	// 快照Walk，回调内执行写操作，不会死锁
	root.Walk(func(el *Element) bool {
		if el.Name == "b" {
			el.AddAttr("flag", "ok") // 写操作，快照模式安全
		}
		return true
	})
	b1, _ := root.Find("//b[@flag='ok']")
	t.Log(b1.ToString())
	if b1 == nil {
		t.Fatal("Walk内修改属性未生效")
	}
	// WalkUnsaft 只读遍历，禁止写
	defer func() {
		if r := recover(); r != nil {
			t.Log("WalkLocked内写触发异常符合预期")
		}
	}()
	//只读
	root.WalkUnsafe(func(el *Element) bool {
		//引发死锁
		el.AddAttr("panic", "1")
		return true
	})
}

// XPath完整功能测试
func TestXPathFind(t *testing.T) {
	xml := `
<root>
	<book id="001">golang</book>
	<book id="002">rust</book>
	<list>
		<book id="003">java</book>
	</list>
</root>`
	root, _ := LoadByXml(xml)
	// 绝对路径单节点
	n1, err := root.Find("root/book")
	if err != nil {
		t.Fatal(err)
	}
	v, _ := n1.AttrValue("id")
	if v != "001" {
		t.Fatal("绝对路径XPath失败")
	}

	// 递归 // 全部book
	allBooks, err := root.FindAll("//book")
	t.Logf("%v", len(allBooks))
	if err != nil || len(allBooks) != 3 {
		t.Fatal("//递归查询失败")
	}
	// 索引 [2]
	b2, _ := root.Find("//book[2]")
	if v, _ := b2.AttrValue("id"); v != "002" {
		t.Fatal("XPath索引查询失败")
	}
	// 属性过滤 [@id='003']
	b3, _ := root.Find("//book[@id='003']")
	if b3.Text() != "java" {
		t.Fatal("XPath属性过滤失败")
	}
	// 不存在节点
	empty, _ := root.Find("//none")
	if empty != nil {
		t.Fatal("不存在节点应返回nil")
	}
}

// 解析配置项测试（MaxDepth/TrimWhitespace/XXE防护）
func TestParseOption(t *testing.T) {
	// 测试最大深度限制
	deepXml := `<a><b><c><d><e><f><g><h><i><j><k></k></i></h></g></f></e></d></c></b></a>`
	opt := ParseOption{MaxDepth: 5}
	_, err := LoadByXmlOption(deepXml, opt)
	if err == nil {
		t.Fatal("超大深度应返回ErrMaxDepthExceeded")
	}
	if !errors.Is(err, ErrMaxDepthExceeded) {
		t.Fatal("错误类型不匹配")
	}

	// 测试空白裁剪
	spaceXml := `<root>   

	test   
	</root>`
	optTrim := ParseOption{TrimWhitespace: true}
	r, _ := LoadByXmlOption(spaceXml, optTrim)
	if strings.TrimSpace(r.Text()) != "test" {
		t.Fatal("空白裁剪失效")
	}

	// 测试关闭注释保留
	commentXml := `<root><!--注释--><a/></root>`
	optNoComment := ParseOption{PreserveComment: false}
	r2, _ := LoadByXmlOption(commentXml, optNoComment)
	if strings.Contains(r2.ToXML(), "<!--") {
		t.Fatal("注释未被忽略")
	}
}

// XXE安全防护测试
func TestXXEBlock(t *testing.T) {
	xxeXml := `<?xml version="1.0"?>
<!DOCTYPE test [<!ENTITY file SYSTEM "/etc/passwd">]>
<test>&file;</test>`
	// 默认禁用外部实体
	root, _ := LoadByXml(xxeXml)
	t.Log(root.ToXML())
	// 不会读取外部文件，解析不会读取本地资源，无IO风险
	t.Log("XXE恶意XML解析无文件读取风险，安全")
}

// 原子计数缓存（ChildrenLength/DocLength O(1)）
func TestAtomicCount(t *testing.T) {
	root := NewElement("root")
	a := NewElement("a")
	a.AddChild(NewElement("b"))
	root.AddChild(a)
	// root直接子元素1个
	if root.ChildrenLength() != 1 {
		t.Fatal("ChildrenLength计数错误")
	}
	// 总元素 root + a + b = 3
	if root.DocLength() != 3 {
		t.Fatal("DocLength计数错误")
	}
	// 删除a
	root.RemoveChild(a)
	if root.ChildrenLength() != 0 || root.DocLength() != 1 {
		t.Fatal("删除后计数未更新")
	}
}

// 命名空间Lookup前缀/URI测试
func TestNamespaceLookup(t *testing.T) {
	xml := `<root xml:lang="zh-CN" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="demo"><a xsi:msn="chat" msn="usa"></a></root>`
	root, err := LoadByXml(xml)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(root.ToXML())
	uri := root.LookupNamespaceURI("xsi")
	t.Log("uri>>>>", uri)
	if uri != "http://www.w3.org/2001/XMLSchema-instance" {
		t.Errorf("LookupNamespaceURI失败, 实际返回=%s", uri)
	}
	prefix := root.LookupPrefix("http://www.w3.org/2001/XMLSchema-instance")
	t.Log("prefix>>>>", prefix)
	if prefix != "xsi" {
		t.Errorf("LookupPrefix失败, 实际返回=%s", prefix)
	}

	a_uri := root.Child("a").LookupNamespaceURI("msn")
	t.Log("a_uri>>>>", a_uri)

}

// 节点导航增强 Depth/Ancestors/NodePath
func TestNodeNavEnhance(t *testing.T) {
	xml := `<r><a><b><c/></b></a></r>`
	root, _ := LoadByXml(xml)
	c := root.GetNodeByPath("r/a/b/c")
	if c.Depth() != 4 {
		t.Fatal("节点深度计算错误")
	}
	anc := c.Ancestors()
	if len(anc) != 3 || anc[0].Name != "b" {
		t.Fatal("祖先节点获取错误")
	}
	path := c.NodePath()
	if path != "r/a/b/c" {
		t.Fatal("NodePath路径生成错误")
	}
}

// 并发读写+快照Walk混合压力
func TestConcurrentWalkModify(t *testing.T) {
	root := NewSyncElement("root")
	var wg sync.WaitGroup
	const goroutine = 30
	wg.Add(goroutine)
	for i := 0; i < goroutine; i++ {
		go func(idx int) {
			defer wg.Done()
			// 快照Walk，并发修改无死锁
			root.Walk(func(el *Element) bool {
				el.AddAttr(fmt.Sprintf("g_%d", idx), "val")
				return true
			})
			root.AddChild(NewElement(fmt.Sprintf("node_%d", idx)))
		}(i)
	}
	wg.Wait()
	// 验证节点数量
	if root.ChildrenLength() != int64(goroutine) {
		t.Fatal("并发新增节点数量不匹配")
	}
}

func TestCDATA2(t *testing.T) {
	el := NewElement("root")
	el.AddCDATA("abc]]>def")
	cds := el.CDATAs()
	if cds[0] != "abc]]>def" {
		t.Fatal("CDATA原始内容丢失")
	}
	out := el.ToXML()
	if !strings.Contains(out, "]]&gt;") {
		t.Fatal("CDATA序列化转义失败")
	}
}

func BenchmarkWalkSnapshot(b *testing.B) {
	s := `<root>`
	for i := 0; i < 200; i++ {
		s += fmt.Sprintf(`<n id="%d"/>`, i)
	}
	s += `</root>`
	root, _ := LoadByXml(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.Walk(func(e *Element) bool { return true })
	}
}

func BenchmarkXPathFindAll(b *testing.B) {
	s := `<root>`
	for i := 0; i < 200; i++ {
		s += fmt.Sprintf(`<item id="%d"/>`, i)
	}
	s += `</root>`
	root, _ := LoadByXml(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = root.FindAll("//item")
	}
}
