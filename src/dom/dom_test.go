package dom

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	fmt.Println()
	fmt.Println("-----------------------------TestAdd---------------------------------")
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.AddNodeByString(`<h>你好</h>`)              //新增节点参数可以为字符串
	el.AddNode(&Element{Name: "g", Value: "新增"}) //新增节点参数为Element
	fmt.Println(el.ToString())
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
	s := `<a Age="10"><b>wu</b><c name="hi">xiao</c><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	//获取节点d 单个值
	fmt.Println("d:", el.Node("d").Value)
	//获取节点d 多个值
	for _, elem := range el.Nodes("d") {
		fmt.Println(elem.Name)
		fmt.Println(elem.Value)
	}
	//获取属性
	for _, attrs := range el.Attrs {
		fmt.Println(attrs.Name)
		fmt.Println(attrs.Value)
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
