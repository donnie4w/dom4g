package dom4g

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAdd---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, err := LoadByXml(s)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	el.AddNodeByString(`<h>你好</h>`)   //新增节点参数可以为字符串
	el.AddNode(NewElement("g", "新增")) //新增节点参数为Element
	fmt.Println(el.ToString())
	g := el.Node("g")
	g.Value = "修改后的值"
	g.AddNode(NewElement("h", ""))
	g.AddAttr("gattr", "gattrv")
	g.Node("h").Value = "我是h"
	fmt.Println(el.SyncToXml())
}

func TestRemove(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestRemove---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.RemoveNode("b")
	fmt.Println(el.ToString())
}

func TestGet(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestGet---------------------------------")
	s := `<a Age="10" go="1"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	//获取节点d 单个值
	fmt.Println("d:", el.Node("d").Value)
	//获取节点d 多个值
	for _, elem := range el.Nodes("d") {
		fmt.Println(elem.Name(), elem.Value)
	}
	//获取属性
	for _, attrs := range el.Attrs {
		fmt.Println(attrs.Name(), attrs.Value)
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

/**
ToString()  & ToXML()
*/
func TestToString_ToXML(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestToString_ToXML---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	f := el.Node("e").Node("f")
	fmt.Println(el.ToString())
	fmt.Println(f.ToString()) //只打印了当前节点的xml信息
	fmt.Println(f.ToXML())    //打印了整个文档的xml信息，与从根节点打印效果的相同
}

/**
ToString()  & ToXML()
*/
func TestParent(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestParent---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	f := el.Node("e").Node("f")
	fmt.Println(f.Parent().ToString()) //打印e节点信息
}

/**
  test length
*/
func TestLength(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestLength---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	e := el.Node("e")
	fmt.Println(el.DocLength())             //打印整个文档节点个数    7
	fmt.Println(el.NodesLength())           //打印子节点个数         5
	fmt.Println(e.NodesLength())            //打印e节点子节点个数     1
	fmt.Println(e.DocLength())              //打印e节点所在整个文档节点个数  7
	el.AddNodeByString(`<h><i>来了吗</i></h>`) //增加2个节点
	newel := NewElement("g", "新增的g值")
	el.AddNode(newel)           //增加1个节点
	fmt.Println(el.DocLength()) //10
	fmt.Println(el.ToXML())
	el.RemoveNode("g") //删除一个节点               // 9
	fmt.Println(el.DocLength())
	el.RemoveNode("h")          //删除一个节点，注意h节点本身包含了一个子节点
	fmt.Println(el.DocLength()) //7
	el.AddNodeByString(`<h><i><j>又来了吗</j><j>来了</j><j>嗯</j></i></h>`)
	fmt.Println(el.DocLength()) //12
	i := el.Node("h").Node("i")
	i.RemoveNode("j")
	fmt.Println(i.DocLength()) // 9
	fmt.Println(i.ToXML())
}

// 2014-11-24
// test GetNodeByPath()
func TestGetByPath(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestGetByPath---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	b := el.GetNodeByPath("a/e/f")
	fmt.Println("a/e/f", b.Name(), ":", b.Value)
	fmt.Println(el.ToString())
	bs := el.GetNodesByPath("a/d")
	fmt.Println(bs[0].Value, bs[1].Value)
}

func TestNew(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestNewElement---------------------------------")
	el := NewElement("a", "a")
	fmt.Println(el.ToString())
	fmt.Println(el.ToXML())
}

func TestCDATA(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestCDATA---------------------------------")
	s := `<a xmlns:android="http://schemas.android.com/apk/res/android"><![CDATA[<b><![CDATA[abc]]></b>]]></a>`
	el, _ := LoadByXml(s)
	b := el.GetNodeByPath("a")
	fmt.Println(el.ToString())
	fmt.Println(b.ToString())
}
