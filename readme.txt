golang 的xml处理库

dom4g提供xml简便的操作方法，如节点 增加，删除，查询，属性增加，修改，删除，查询等功能

具体操作可以见测试文件dom_test.go

func TestAdd(t *testing.T) {
	fmt.Println()
	fmt.Println("-------------------------TestAdd-----------------------------")
	s := `<a Age="10"><d>dong</d><d>wxd</d><e><f>hello</f></e></a>`
	el, _ := LoadByXml(s)
	el.AddNodeByString(`<h>你好</h>`)              //新增节点参数可以为字符串
	el.AddNode(&Element{Name: "g", Value: "新增"}) //新增节点参数为Element
	fmt.Println(el.ToString()) 
	//控制台打印
	//<a Age="10"><d>dong</d><d>wxd</d><e><f>hello</f></e><h>你好</h><g>新增</g></a>
}