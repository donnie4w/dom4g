golang 的xml处理库
dom4g提供xml简便的操作方法，如节点 增加，删除，查询，属性增加，修改，删除，查询等功能

方法简单介绍：

导入xml文档：返回Element指针
1，LoadByStream  
2，LoadByXml   参数为字符串

创建节点
1，LoadByStream
2，LoadByXml
3，NewElement   返回指定名字与值的Element指针

转字符串输出
1，ToString   当前节点xml字符串
2，ToXml      整个文档xml字符串
3，SyncToXml  输出整个文档xml字符串，为同步方法，加锁对所有节点都会起作用
4，DocLength  整个文档的节点数 

获取节点名字，值，属性
1，获取Element的Name()，Value，Attrs(属性集合)

属性操作
1，AttrValue  返回指定名字的属性的值
2，AddAttr    给当前节点增加一个指定名字与值的属性
3，RemoveAttr  删除指定名字的属性

子节点操作
1，Node  返回指定名字的Element子节点
2，Nodes 返回指定名字的Element 集合
3，NodesLength  返回子节点个数
4，AllNodes  返回所有子节点集合
5，RemoveNode 删除指定名字的子节点(可能有多个相同名字的节点，将都被删除)
6，AddNode  增加一个子节点
7，AddNodeByString  增加一个子节点，参数为字符串如：<a>b</a>  结构需为xml结构

获取父节点
1，Parent  返回父节点Element指针，若当前节点为根节点，则返回nil
 

具体操作可以见测试文件dom_test.go