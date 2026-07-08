// Copyright (c) 2016, donnie <donnie4w@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// github.com/donnie4w/dom4g

package dom4g

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type NodeType int

const (
	NodeElement NodeType = iota
	NodeText
	NodeCDATA
	NodeComment
	NodeProcInst
)

// XML 命名空间标准 URI
const (
	XMLNamespaceURI   = "http://www.w3.org/XML/1998/namespace"
	XMLNSNamespaceURI = "http://www.w3.org/2000/xmlns/"
)

const (
	DefaultMaxDepth   = 1000 // 默认最大解析深度，防栈溢出
	defaultBuilderCap = 256  // Builder 池默认容量
)

var (
	ErrNilNode          = errors.New("dom4g: node is nil")
	ErrNotFound         = errors.New("dom4g: reference node not found")
	ErrInvalidName      = errors.New("dom4g: invalid element name")
	ErrParse            = errors.New("dom4g: xml parse error")
	ErrMaxDepthExceeded = errors.New("dom4g: max parse depth exceeded")
	ErrXXEBlocked       = errors.New("dom4g: external entity blocked for security")
	ErrInvalidXPath     = errors.New("dom4g: invalid xpath expression")
	ErrUnsupportedXPath = errors.New("dom4g: unsupported xpath feature")
)

// ParseError 解析错误，携带位置信息
type ParseError struct {
	Line int
	Col  int
	Err  error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%v at line %d, col %d", e.Err, e.Line, e.Col)
	}
	return e.Err.Error()
}

func (e *ParseError) Unwrap() error { return e.Err }

// ParseOption XML 解析配置
type ParseOption struct {
	MaxDepth        int  // 最大递归深度，<=0 使用默认值
	TrimWhitespace  bool // 是否裁剪纯空白文本节点
	DisableXXE      bool // 是否禁用外部实体（默认开启）
	PreserveComment bool // 是否保留注释节点
}

var defaultParseOption = ParseOption{
	MaxDepth:        DefaultMaxDepth,
	TrimWhitespace:  true,
	DisableXXE:      true,
	PreserveComment: true,
}

// Node DOM 节点通用接口
type Node interface {
	// Type 基础信息
	Type() NodeType
	ToString() string // 等同于 OuterXML
	InnerText() string

	// Parent 树结构
	Parent() *Element
	Root() *Element
	PreviousSibling() Node
	NextSibling() Node

	//Clone 克隆与删除
	Clone(deep bool) Node
	Remove() bool

	// 内部方法
	setParent(*Element)
	ownerDoc() *Element
}

// 基础节点公共字段，减少重复代码
type baseNode struct {
	parent *Element
}

func (n *baseNode) Parent() *Element     { return n.parent }
func (n *baseNode) setParent(p *Element) { n.parent = p }
func (n *baseNode) ownerDoc() *Element {
	if n.parent == nil {
		return nil
	}
	return n.parent.root
}
func (n *baseNode) Root() *Element {
	if n.parent == nil {
		return nil
	}
	return n.parent.root
}

// Attr XML 属性
type Attr struct {
	Space string // 命名空间 URI
	Name  string // 完整名称（带前缀：prefix:local）
	Value string // 属性值
}

// TextNode 文本节点
type TextNode struct {
	baseNode
	Value string
}

func (t *TextNode) Type() NodeType        { return NodeText }
func (t *TextNode) ToString() string      { return xmlEscape(t.Value, false) }
func (t *TextNode) InnerText() string     { return t.Value }
func (t *TextNode) Remove() bool          { return removeNodeFromParent(t) }
func (t *TextNode) PreviousSibling() Node { return prevSiblingNode(t) }
func (t *TextNode) NextSibling() Node     { return nextSiblingNode(t) }
func (t *TextNode) Clone(deep bool) Node {
	return &TextNode{Value: t.Value}
}

// CDATANode CDATA 段节点
type CDATANode struct {
	baseNode
	Value string
}

func (c *CDATANode) Type() NodeType    { return NodeCDATA }
func (c *CDATANode) InnerText() string { return c.Value }
func (c *CDATANode) ToString() string {
	safe := strings.ReplaceAll(c.Value, "]]>", "]]&gt;")
	return fmt.Sprintf("<![CDATA[%s]]>", safe)
}
func (c *CDATANode) Remove() bool          { return removeNodeFromParent(c) }
func (c *CDATANode) PreviousSibling() Node { return prevSiblingNode(c) }
func (c *CDATANode) NextSibling() Node     { return nextSiblingNode(c) }
func (c *CDATANode) Clone(deep bool) Node {
	return &CDATANode{Value: c.Value}
}

// CommentNode 注释节点
type CommentNode struct {
	baseNode
	Text string
}

func (c *CommentNode) Type() NodeType        { return NodeComment }
func (c *CommentNode) InnerText() string     { return "" }
func (c *CommentNode) ToString() string      { return "<!--" + c.Text + "-->" }
func (c *CommentNode) Remove() bool          { return removeNodeFromParent(c) }
func (c *CommentNode) PreviousSibling() Node { return prevSiblingNode(c) }
func (c *CommentNode) NextSibling() Node     { return nextSiblingNode(c) }
func (c *CommentNode) Clone(deep bool) Node {
	return &CommentNode{Text: c.Text}
}

// ProcInstNode 处理指令节点
type ProcInstNode struct {
	baseNode
	Target string
	Data   string
}

func (p *ProcInstNode) Type() NodeType    { return NodeProcInst }
func (p *ProcInstNode) InnerText() string { return "" }
func (p *ProcInstNode) ToString() string {
	return fmt.Sprintf(`<?%s %s?>`, p.Target, p.Data)
}
func (p *ProcInstNode) Remove() bool          { return removeNodeFromParent(p) }
func (p *ProcInstNode) PreviousSibling() Node { return prevSiblingNode(p) }
func (p *ProcInstNode) NextSibling() Node     { return nextSiblingNode(p) }
func (p *ProcInstNode) Clone(deep bool) Node {
	return &ProcInstNode{Target: p.Target, Data: p.Data}
}

// Element XML 元素节点，实现 Node 接口
type Element struct {
	head   string // XML 头声明（仅根节点有效）
	Space  string // 命名空间 URI
	Prefix string // 命名空间前缀
	Name   string // 节点本地名

	Attrs      []*Attr               // 属性列表
	childs     []Node                // 所有子节点
	parent     *Element              // 父节点
	elementmap map[string][]*Element // 子元素按名索引，懒初始化
	attrmap    map[string]string     // 属性快速索引，懒初始化

	lc     *sync.RWMutex // 根节点统一读写锁
	root   *Element      // 根节点指针
	isSync uint32        // 并发安全开关，原子操作，仅根节点有效

	childCount int64 // 直接子元素数量
	descCount  int64 // 后代元素总数（含自身）

	spaceNS map[string]string //
}

func (t *Element) Type() NodeType       { return NodeElement }
func (t *Element) Parent() *Element     { return t.parent }
func (t *Element) setParent(p *Element) { t.parent = p }
func (t *Element) ownerDoc() *Element   { return t.root }
func (t *Element) Root() *Element       { return t.root }

func (t *Element) ToString() string { return t.OuterXML() }

func (t *Element) Remove() bool {
	if t.parent == nil {
		return false
	}
	return t.parent.RemoveChild(t)
}

func (t *Element) PreviousSibling() Node {
	if t.parent == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	idx := t.parent.findNodeIndex(t)
	if idx <= 0 {
		return nil
	}
	return t.parent.childs[idx-1]
}

func (t *Element) NextSibling() Node {
	if t.parent == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	idx := t.parent.findNodeIndex(t)
	if idx == -1 || idx >= len(t.parent.childs)-1 {
		return nil
	}
	return t.parent.childs[idx+1]
}

func (t *Element) Clone(deep bool) Node {
	return t.CloneElement(deep)
}

func (t *Element) rLock() {
	if t == nil {
		return
	}
	root := t.root
	if root == nil {
		return
	}
	if atomic.LoadUint32(&root.isSync) == 1 {
		root.lc.RLock()
	}
}

func (t *Element) rUnlock() {
	if t == nil {
		return
	}
	root := t.root
	if root == nil {
		return
	}
	if atomic.LoadUint32(&root.isSync) == 1 {
		root.lc.RUnlock()
	}
}

func (t *Element) wLock() {
	if t == nil {
		return
	}
	root := t.root
	if root == nil {
		return
	}
	if atomic.LoadUint32(&root.isSync) == 1 {
		root.lc.Lock()
	}
}

func (t *Element) wUnlock() {
	if t == nil {
		return
	}
	root := t.root
	if root == nil {
		return
	}
	if atomic.LoadUint32(&root.isSync) == 1 {
		root.lc.Unlock()
	}
}

// EnableSync 开启整棵树并发安全模式
func (t *Element) EnableSync() {
	if t == nil {
		return
	}
	root := t.root
	atomic.StoreUint32(&root.isSync, 1)
}

// DisableSync 关闭并发安全模式
func (t *Element) DisableSync() {
	if t == nil {
		return
	}
	root := t.root
	atomic.StoreUint32(&root.isSync, 0)
}

// NewElement 创建新元素节点
func NewElement(elementName string) *Element {
	el := &Element{
		Name:       elementName,
		childs:     make([]Node, 0),
		elementmap: nil, // 懒初始化
		attrmap:    nil, // 懒初始化
		lc:         new(sync.RWMutex),
		isSync:     0,
		childCount: 0,
		descCount:  1,
	}
	el.root = el
	return el
}

// NewSyncElement 创建默认开启并发安全的元素
func NewSyncElement(elementName string) *Element {
	el := NewElement(elementName)
	el.EnableSync()
	return el
}

// LoadByXml 从字符串加载 XML（默认配置）
func LoadByXml(xmlstr string) (*Element, error) {
	return LoadByXmlOption(xmlstr, defaultParseOption)
}

// LoadByXmlOption 带配置加载 XML
func LoadByXmlOption(xmlstr string, opt ParseOption) (*Element, error) {
	return loadByStream(strings.NewReader(xmlstr), nil, opt)
}

// LoadByStream 从流加载 XML（默认配置）
func LoadByStream(r io.Reader) (*Element, error) {
	return LoadByStreamOption(r, defaultParseOption)
}

// LoadByStreamOption 带配置从流加载 XML
func LoadByStreamOption(r io.Reader, opt ParseOption) (*Element, error) {
	return loadByStream(r, nil, opt)
}

// 核心解析函数
func loadByStream(r io.Reader, cdatas []string, opt ParseOption) (root *Element, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = &ParseError{
				Err: fmt.Errorf("%w: %v\n%s", ErrParse, rec, debug.Stack()),
			}
		}
	}()

	if opt.MaxDepth <= 0 {
		opt.MaxDepth = DefaultMaxDepth
	}

	decoder := xml.NewDecoder(r)
	decoder.Strict = true

	// 安全防护：禁用外部实体
	if opt.DisableXXE {
		decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
			return input, nil
		}
	}

	var current *Element
	isRoot := true
	head := ""
	currentDepth := 0

	// 命名空间栈
	var nsStack []map[string]string
	nsStack = append(nsStack, map[string]string{
		"xml":   XMLNamespaceURI,
		"xmlns": XMLNSNamespaceURI,
	})

	for t, er := decoder.Token(); er == nil; t, er = decoder.Token() {

		switch token := t.(type) {
		case xml.StartElement:
			currentDepth++
			if currentDepth > opt.MaxDepth {
				return nil, &ParseError{
					Line: int(decoder.InputOffset()),
					Err:  ErrMaxDepthExceeded,
				}
			}

			// 命名空间处理
			parentNS := nsStack[len(nsStack)-1]
			currentNS := make(map[string]string, len(parentNS)+2)
			for k, v := range parentNS {
				currentNS[k] = v
			}

			el := &Element{
				Space:      token.Name.Space,
				Name:       token.Name.Local,
				childs:     make([]Node, 0),
				elementmap: nil,
				attrmap:    nil,
				lc:         new(sync.RWMutex),
				isSync:     0,
				childCount: 0,
				descCount:  1,
			}

			spaceNS := make(map[string]string, 0)

			for _, a := range token.Attr {
				for _, v := range currentNS {
					if a.Name.Space != v {
						currentNS[a.Name.Local] = a.Value
					}
				}
				if a.Name.Space == "xmlns" {
					spaceNS[a.Name.Local] = a.Value
				} else if a.Name.Space != "" && a.Name.Space != "xmlns" {
					spaceNS[a.Name.Local] = a.Name.Space
				}

			}
			el.spaceNS = spaceNS

			// 反查元素前缀
			if token.Name.Space != "" {
				for prefix, uri := range currentNS {
					if uri == token.Name.Space && prefix != "" {
						el.Prefix = prefix
						break
					}
				}
			}

			// 处理属性
			for _, a := range token.Attr {
				ar := &Attr{
					Space: a.Name.Space,
					Value: a.Value,
				}

				switch {
				case a.Name.Local == "xmlns" && a.Name.Space == "":
					ar.Name = "xmlns"
				case a.Name.Space == XMLNSNamespaceURI || a.Name.Space == "xmlns":
					ar.Name = "xmlns:" + a.Name.Local
				case a.Name.Space != "":
					ar.Name = a.Name.Local
					for prefix, uri := range currentNS {
						if uri == a.Name.Space && prefix != "" {
							ar.Name = prefix + ":" + a.Name.Local
							spaceNS[ar.Name] = a.Name.Space
							break
						}
					}
				default:
					ar.Name = a.Name.Local
				}

				el.appendAttr(ar)
			}

			nsStack = append(nsStack, currentNS)

			if isRoot {
				isRoot = false
				el.root = el
				current = el
			} else {
				current.appendChild(el)
			}
			current = el

		case xml.EndElement:
			if len(nsStack) > 1 {
				nsStack = nsStack[:len(nsStack)-1]
			}
			currentDepth--
			if current.parent != nil {
				current = current.parent
			}

		case xml.CharData:
			text := string(token.Copy())
			if opt.TrimWhitespace && strings.TrimSpace(text) == "" {
				break
			}

			if len(cdatas) > 0 {
				segments := splitCDATAPlaceholder(text)
				for _, seg := range segments {
					if seg.IsPlaceholder {
						if idx, err := strconv.Atoi(seg.Index); err == nil && idx >= 0 && idx < len(cdatas) {
							current.appendChild(&CDATANode{Value: cdatas[idx]})
						}
					} else {
						current.appendChild(&TextNode{Value: seg.Text})
					}
				}
			} else {
				current.appendChild(&TextNode{Value: text})
			}

		case xml.Comment:
			if opt.PreserveComment && current != nil {
				current.appendChild(&CommentNode{Text: string(token.Copy())})
			}

		case xml.ProcInst:
			pi := token.Copy()
			if strings.EqualFold(pi.Target, "xml") {
				head = fmt.Sprintf(`<?%s %s?>`, pi.Target, string(pi.Inst))
			} else if current != nil {
				current.appendChild(&ProcInstNode{
					Target: pi.Target,
					Data:   string(pi.Inst),
				})
			}

		case xml.Directive:
			// DTD 默认忽略，安全起见不处理外部实体
		default:
			// 未知类型静默跳过
		}
	}

	if current != nil {
		current.root.head = head
	}
	return current, nil
}

// ===================== 内部工具方法 =====================

// 追加属性，懒初始化 map
func (t *Element) appendAttr(a *Attr) {
	t.Attrs = append(t.Attrs, a)
	if t.attrmap == nil {
		t.attrmap = make(map[string]string)
	}
	t.attrmap[a.Name] = a.Value
}

// 追加子节点，维护索引与计数
func (t *Element) appendChild(node Node) {
	node.setParent(t)
	t.childs = append(t.childs, node)

	if el, ok := node.(*Element); ok {
		if t.elementmap == nil {
			t.elementmap = make(map[string][]*Element)
		}
		t.elementmap[el.Name] = append(t.elementmap[el.Name], el)
		el.root = t.root
		el.lc = t.lc
		el.syncRoot(t.root)

		atomic.AddInt64(&t.childCount, 1)
		p := t
		for p != nil {
			atomic.AddInt64(&p.descCount, el.descCount)
			p = p.parent
		}
	}
}

// 查找节点在 childs 中的索引
func (t *Element) findNodeIndex(n Node) int {
	for i, child := range t.childs {
		if child == n {
			return i
		}
	}
	return -1
}

// 从索引表移除元素
func (t *Element) removeFromMap(el *Element) {
	if t.elementmap == nil {
		return
	}
	if els, ok := t.elementmap[el.Name]; ok {
		for i, e := range els {
			if e == el {
				t.elementmap[el.Name] = append(els[:i], els[i+1:]...)
				break
			}
		}
		if len(t.elementmap[el.Name]) == 0 {
			delete(t.elementmap, el.Name)
		}
	}
}

// 接管子节点
func (t *Element) adoptChild(el *Element) {
	// 从原父节点移除
	if el.parent != nil {
		el.parent.RemoveChild(el)
	}
	el.parent = t
	el.root = t.root
	el.lc = t.root.lc
	el.syncRoot(t.root)
}

// 递归同步根指针与锁
func (t *Element) syncRoot(root *Element) {
	t.lc = root.lc
	t.root = root
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			el.syncRoot(root)
		}
	}
}

// 收集文本
func (t *Element) collectText(sb *strings.Builder) {
	for _, child := range t.childs {
		switch c := child.(type) {
		case *TextNode:
			sb.WriteString(c.Value)
		case *CDATANode:
			sb.WriteString(c.Value)
		case *Element:
			c.collectText(sb)
		}
	}
}

// 通用节点移除辅助函数
func removeNodeFromParent(n Node) bool {
	parent := n.Parent()
	if parent == nil {
		return false
	}
	parent.wLock()
	defer parent.wUnlock()

	idx := parent.findNodeIndex(n)
	if idx == -1 {
		return false
	}

	parent.childs = append(parent.childs[:idx], parent.childs[idx+1:]...)

	if el, ok := n.(*Element); ok {
		parent.removeFromMap(el)
		el.parent = nil
		el.syncRoot(el)

		// 更新计数
		atomic.AddInt64(&parent.childCount, -1)
		p := parent
		for p != nil {
			atomic.AddInt64(&p.descCount, -el.descCount)
			p = p.parent
		}
	}

	n.setParent(nil)
	return true
}

// 兄弟节点查找辅助
func prevSiblingNode(n Node) Node {
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	parent.rLock()
	defer parent.rUnlock()
	idx := parent.findNodeIndex(n)
	if idx <= 0 {
		return nil
	}
	return parent.childs[idx-1]
}

func nextSiblingNode(n Node) Node {
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	parent.rLock()
	defer parent.rUnlock()
	idx := parent.findNodeIndex(n)
	if idx == -1 || idx >= len(parent.childs)-1 {
		return nil
	}
	return parent.childs[idx+1]
}

// Builder 对象池，减少 GC
var builderPool = sync.Pool{
	New: func() interface{} {
		sb := &strings.Builder{}
		sb.Grow(defaultBuilderCap)
		return sb
	},
}

func acquireBuilder() *strings.Builder {
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

func releaseBuilder(sb *strings.Builder) {
	builderPool.Put(sb)
}

// OuterXML 当前节点完整 XML（含自身标签）
func (t *Element) OuterXML() string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()
	sb := acquireBuilder()
	defer releaseBuilder(sb)
	t.toXML(sb, "", "")
	return sb.String()
}

// InnerXML 子节点 XML（不含自身标签）
func (t *Element) InnerXML() string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()
	sb := acquireBuilder()
	defer releaseBuilder(sb)
	for _, child := range t.childs {
		sb.WriteString(child.ToString())
	}
	return sb.String()
}

// ToXML 完整文档 XML（从根节点）
func (t *Element) ToXML() string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()
	root := t.root
	sb := acquireBuilder()
	defer releaseBuilder(sb)

	if root.head != "" {
		sb.WriteString(root.head)
		sb.WriteByte('\n')
	}
	root.toXML(sb, "", "")
	return sb.String()
}

// ToXMLIndent 格式化缩进输出
func (t *Element) ToXMLIndent(prefix, indent string) string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()
	root := t.root
	sb := acquireBuilder()
	defer releaseBuilder(sb)

	if root.head != "" {
		sb.WriteString(root.head)
		sb.WriteByte('\n')
	}
	root.toXML(sb, indent, prefix)
	return sb.String()
}

// SyncToXml 并发安全输出完整文档
func (t *Element) SyncToXml() string {
	return t.ToXML()
}

// 核心序列化方法
func (t *Element) toXML(sb *strings.Builder, indent, curIndent string) {
	tagName := t.Name
	if t.Prefix != "" {
		tagName = t.Prefix + ":" + tagName
	}

	sb.WriteString(curIndent)
	sb.WriteByte('<')
	sb.WriteString(tagName)

	// 写入属性
	for _, attr := range t.Attrs {
		sb.WriteByte(' ')
		sb.WriteString(attr.Name)
		sb.WriteString(`="`)
		sb.WriteString(xmlEscape(attr.Value, true))
		sb.WriteByte('"')
	}

	hasContent := len(t.childs) > 0
	if !hasContent {
		sb.WriteString("/>")
		if indent != "" {
			sb.WriteByte('\n')
		}
		return
	}

	sb.WriteByte('>')
	if indent != "" {
		sb.WriteByte('\n')
	}
	nextIndent := curIndent + indent

	for _, child := range t.childs {
		switch c := child.(type) {
		case *Element:
			c.toXML(sb, indent, nextIndent)
		case *TextNode, *CDATANode, *CommentNode, *ProcInstNode:
			if indent != "" {
				sb.WriteString(nextIndent)
			}
			sb.WriteString(c.ToString())
			if indent != "" {
				sb.WriteByte('\n')
			}
		}
	}

	if indent != "" {
		sb.WriteString(curIndent)
	}
	sb.WriteString("</")
	sb.WriteString(tagName)
	sb.WriteByte('>')
	if indent != "" {
		sb.WriteByte('\n')
	}
}

// Head 获取 XML 头声明
func (t *Element) Head() string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()
	return t.root.head
}

// ===================== 节点查询 API =====================

// Child 获取第一个同名直接子元素
func (t *Element) Child(name string) *Element {
	if t == nil || t.elementmap == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	if els, ok := t.elementmap[name]; ok && len(els) > 0 {
		return els[0]
	}
	return nil
}

// Children 获取所有同名直接子元素
func (t *Element) Children(name string) []*Element {
	if t == nil || t.elementmap == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	els, ok := t.elementmap[name]
	if !ok {
		return nil
	}
	ret := make([]*Element, len(els))
	copy(ret, els)
	return ret
}

// AllChildren 获取所有直接子元素
func (t *Element) AllChildren() []*Element {
	if t == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	var ret []*Element
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			ret = append(ret, el)
		}
	}
	return ret
}

// FirstChild 第一个子元素
func (t *Element) FirstChild() *Element {
	if t == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			return el
		}
	}
	return nil
}

// LastChild 最后一个子元素
func (t *Element) LastChild() *Element {
	if t == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	for i := len(t.childs) - 1; i >= 0; i-- {
		if el, ok := t.childs[i].(*Element); ok {
			return el
		}
	}
	return nil
}

// GetElementsByTagName 递归查找所有同名元素
func (t *Element) GetElementsByTagName(name string) []*Element {
	if t == nil {
		return nil
	}
	var res []*Element
	t.Walk(func(el *Element) bool {
		if el.Name == name {
			res = append(res, el)
		}
		return true
	})
	return res
}

// GetNodeByPath 按路径查找单个节点（支持相对/绝对路径）
func (t *Element) GetNodeByPath(path string) *Element {
	if t == nil || path == "" {
		return nil
	}
	t.rLock()
	defer t.rUnlock()

	paths := strings.Split(path, "/")
	if len(paths) == 0 {
		return nil
	}

	e := t
	if paths[0] == e.Name {
		paths = paths[1:]
	}

	for _, p := range paths {
		e = e.Child(p)
		if e == nil {
			return nil
		}
	}
	return e
}

// GetNodesByPath 按路径查找所有节点
func (t *Element) GetNodesByPath(path string) []*Element {
	if t == nil || path == "" {
		return nil
	}
	t.rLock()
	defer t.rUnlock()

	paths := strings.Split(path, "/")
	length := len(paths)
	if length == 0 {
		return nil
	}
	if length == 1 {
		return t.Children(paths[0])
	}

	parentPath := strings.Join(paths[:length-1], "/")
	parent := t.GetNodeByPath(parentPath)
	if parent == nil {
		return nil
	}
	return parent.Children(paths[length-1])
}

// ===================== XPath 支持 =====================

// Find 通过 XPath 查找单个元素
// 支持：/a/b/c、//title、book[1]、book[@id='1001']、//book[@id]/title
func (t *Element) Find(xpath string) (*Element, error) {
	nodes, err := t.FindAll(xpath)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// FindAll 通过 XPath 查找所有匹配元素
func (t *Element) FindAll(xpath string) ([]*Element, error) {
	if t == nil || xpath == "" {
		return nil, ErrInvalidXPath
	}
	t.rLock()
	defer t.rUnlock()

	tokens, err := parseXPath(xpath)
	if err != nil {
		return nil, err
	}
	fmt.Println(tokens)
	current := []*Element{t}
	for i, step := range tokens {
		var next []*Element

		// 递归下降 //
		if step.Recursive {
			next = nil
			// 遍历当前所有上下文节点，收集自身+全部后代
			for _, node := range current {
				descendants := collectAllDescendants(node)
				next = append(next, descendants...)
			}
			// 无名称 //* 直接全量返回
			if step.Name == "" {
				current = next
				continue
			}
			// 过滤匹配名称节点
			var filtered []*Element
			for _, n := range next {
				if n.Name == step.Name {
					filtered = append(filtered, n)
				}
			}
			next = filtered
		} else {
			// 相对路径，第一步匹配当前节点名
			if i == 0 && step.Name == t.Name {
				next = []*Element{t}
			} else {
				for _, node := range current {
					kids := node.Children(step.Name)
					next = append(next, kids...)
				}
			}
		}

		// 索引过滤 [n]
		if step.Index > 0 {
			if step.Index-1 < len(next) {
				next = []*Element{next[step.Index-1]}
			} else {
				next = nil
			}
		}

		// 属性过滤 [@k='v']
		if step.AttrKey != "" {
			var filtered []*Element
			for _, n := range next {
				v, ok := n.AttrValue(step.AttrKey)
				if ok && v == step.AttrValue {
					filtered = append(filtered, n)
				}
			}
			next = filtered
		}

		current = next
		if len(current) == 0 {
			break
		}
	}

	return current, nil
}

// XPath 步骤结构
type xpathStep struct {
	Name      string
	Recursive bool   // 是否递归
	Index     int    // 索引，0表示无
	AttrKey   string // 属性过滤键
	AttrValue string // 属性过滤值
}

// 解析 XPath 表达式（子集）
func parseXPath(xpath string) ([]xpathStep, error) {
	if strings.ContainsAny(xpath, "()*|+") {
		return nil, fmt.Errorf("%w: function/wildcard/union not supported", ErrUnsupportedXPath)
	}

	// 处理 // 递归标记
	xpath = strings.TrimSpace(xpath)
	parts := strings.Split(xpath, "/")
	var steps []xpathStep

	i := 0
	for i < len(parts) {
		seg := parts[i]
		fmt.Println("seq>>>", seg)
		if seg == "" {
			// 连续 // 表示递归
			if i+1 < len(parts) {
				i++
				name, idx, attrKey, attrVal, err := parseStepSegment(parts[i])
				if err != nil {
					return nil, err
				}
				steps = append(steps, xpathStep{
					Name:      name,
					Recursive: true,
					Index:     idx,
					AttrKey:   attrKey,
					AttrValue: attrVal,
				})
			}
			i++
			continue
		}

		name, idx, attrKey, attrVal, err := parseStepSegment(seg)
		if err != nil {
			return nil, err
		}
		steps = append(steps, xpathStep{
			Name:      name,
			Index:     idx,
			AttrKey:   attrKey,
			AttrValue: attrVal,
		})
		i++
	}

	return steps, nil
}

// 解析单步片段：name[index] 或 name[@k='v']
func parseStepSegment(seg string) (name string, index int, attrKey, attrVal string, err error) {
	bracketIdx := strings.Index(seg, "[")
	if bracketIdx == -1 {
		return seg, 0, "", "", nil
	}

	name = seg[:bracketIdx]
	inside := seg[bracketIdx+1 : len(seg)-1]

	if strings.HasPrefix(inside, "@") {
		// 属性过滤
		eqIdx := strings.Index(inside, "=")
		if eqIdx == -1 {
			return "", 0, "", "", fmt.Errorf("%w: invalid attribute filter", ErrInvalidXPath)
		}
		attrKey = strings.TrimSpace(inside[1:eqIdx])
		attrVal = strings.Trim(inside[eqIdx+1:], "'\"")
		return name, 0, attrKey, attrVal, nil
	}

	// 索引
	idx, err := strconv.Atoi(strings.TrimSpace(inside))
	if err != nil {
		return "", 0, "", "", fmt.Errorf("%w: invalid index", ErrInvalidXPath)
	}
	return name, idx, "", "", nil
}

// 收集所有后代元素
func collectAllDescendants(el *Element) []*Element {
	var result []*Element
	result = append(result, el)
	for _, child := range el.childs {
		if e, ok := child.(*Element); ok {
			result = append(result, collectAllDescendants(e)...)
		}
	}
	return result
}

// ===================== 遍历 API =====================

// Walk 深度优先遍历所有元素
// 回调内可安全执行读写操作，遍历期间树结构变更不影响遍历结果
func (t *Element) Walk(fn func(*Element) bool) {
	if t == nil || fn == nil {
		return
	}

	t.rLock()
	var snapshot []*Element
	t.walkCollect(&snapshot)
	t.rUnlock()

	for _, el := range snapshot {
		if !fn(el) {
			break
		}
	}
}

// WalkUnsafe 持有锁遍历（高性能只读场景，回调内禁止写操作）
// 警告：回调内执行写操作会导致死锁
func (t *Element) WalkUnsafe(fn func(*Element) bool) {
	if t == nil || fn == nil {
		return
	}
	t.rLock()
	defer t.rUnlock()
	t.walk(fn)
}

// 内部递归收集
func (t *Element) walkCollect(out *[]*Element) {
	*out = append(*out, t)
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			el.walkCollect(out)
		}
	}
}

// 内部递归遍历（持有锁）
func (t *Element) walk(fn func(*Element) bool) bool {
	if !fn(t) {
		return false
	}
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			if !el.walk(fn) {
				return false
			}
		}
	}
	return true
}

// ===================== 属性操作 =====================

// AttrValue 获取属性值
func (t *Element) AttrValue(name string) (string, bool) {
	if t == nil || t.attrmap == nil {
		return "", false
	}
	t.rLock()
	defer t.rUnlock()
	v, ok := t.attrmap[name]
	return v, ok
}

// HasAttr 判断属性是否存在
func (t *Element) HasAttr(name string) bool {
	_, ok := t.AttrValue(name)
	return ok
}

// AddAttr 新增或覆盖属性
func (t *Element) AddAttr(name, value string) *Element {
	if t == nil {
		return t
	}
	t.wLock()
	defer t.wUnlock()

	if t.attrmap == nil {
		t.attrmap = make(map[string]string)
	}

	if _, ok := t.attrmap[name]; ok {
		// 覆盖
		for _, v := range t.Attrs {
			if v.Name == name {
				v.Value = value
				break
			}
		}
	} else {
		t.Attrs = append(t.Attrs, &Attr{Name: name, Value: value})
	}
	t.attrmap[name] = value
	return t
}

// RemoveAttr 删除属性
func (t *Element) RemoveAttr(name string) bool {
	if t == nil || t.attrmap == nil {
		return false
	}
	t.wLock()
	defer t.wUnlock()

	if _, ok := t.attrmap[name]; !ok {
		return false
	}
	delete(t.attrmap, name)

	newAttrs := make([]*Attr, 0, len(t.Attrs)-1)
	for _, v := range t.Attrs {
		if v.Name != name {
			newAttrs = append(newAttrs, v)
		}
	}
	t.Attrs = newAttrs
	return true
}

// ===================== 文本与 CDATA =====================

// Text 返回当前节点下的文本数据
func (t *Element) Text() string {
	t.rLock()
	defer t.rUnlock()
	var sb strings.Builder
	for _, child := range t.childs {
		switch c := child.(type) {
		case *TextNode:
			sb.WriteString(c.Value)
		}
	}
	return sb.String()
}

// InnerText 返回所有子节点文本
func (t *Element) InnerText() string {
	t.rLock()
	defer t.rUnlock()
	var sb strings.Builder
	t.collectText(&sb)
	return sb.String()
}

// AddText 添加文本节点
func (t *Element) AddText(content string) *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	if strings.TrimSpace(content) == "" {
		return t
	}
	t.appendChild(&TextNode{Value: content})
	return t
}

// SetText 重置直接子文本
func (t *Element) SetText(content string) *Element {
	t.ClearTexts()
	if strings.TrimSpace(content) == "" {
		return t
	}
	return t.AddText(content)
}

// ClearTexts 清空直接子文本节点
func (t *Element) ClearTexts() *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	var newChilds []Node
	for _, n := range t.childs {
		if _, ok := n.(*TextNode); !ok {
			newChilds = append(newChilds, n)
		}
	}
	t.childs = newChilds
	return t
}

// AddCDATA 添加 CDATA
func (t *Element) AddCDATA(content string) *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	t.appendChild(&CDATANode{Value: content})
	return t
}

// SetCDATA 重置 CDATA
func (t *Element) SetCDATA(raw string) *Element {
	t.ClearCDATAs()
	return t.AddCDATA(raw)
}

// ClearCDATAs 清空直接子 CDATA
func (t *Element) ClearCDATAs() *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	var newChilds []Node
	for _, n := range t.childs {
		if _, ok := n.(*CDATANode); !ok {
			newChilds = append(newChilds, n)
		}
	}
	t.childs = newChilds
	return t
}

// CDATAs 获取所有后代 CDATA 内容
func (t *Element) CDATAs() []string {
	if t == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	var res []string
	t.collectCDATA(&res)
	return res
}

func (t *Element) collectCDATA(out *[]string) {
	for _, n := range t.childs {
		switch c := n.(type) {
		case *CDATANode:
			*out = append(*out, c.Value)
		case *Element:
			c.collectCDATA(out)
		}
	}
}

// AddComment 添加注释
func (t *Element) AddComment(text string) *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	t.appendChild(&CommentNode{Text: text})
	return t
}

// AddProcInst 添加处理指令
func (t *Element) AddProcInst(target, data string) *Element {
	if t == nil {
		return nil
	}
	t.wLock()
	defer t.wUnlock()
	t.appendChild(&ProcInstNode{Target: target, Data: data})
	return t
}

// ===================== 节点增删改 =====================

// AddChild 添加子元素
func (t *Element) AddChild(el *Element) error {
	if t == nil {
		return ErrNilNode
	}
	t.wLock()
	defer t.wUnlock()
	if el == nil {
		return ErrNilNode
	}
	if el.Name == "" {
		return ErrInvalidName
	}
	t.adoptChild(el)
	t.appendChild(el)
	return nil
}

// AddChildByString 通过 XML 字符串添加子元素
func (t *Element) AddChildByString(xmlstr string) error {
	if t == nil {
		return ErrNilNode
	}
	el, err := LoadByXml(xmlstr)
	if err != nil {
		return err
	}
	return t.AddChild(el)
}

// InsertBefore 在引用节点前插入
func (t *Element) InsertBefore(newEl, refEl *Element) error {
	if t == nil {
		return ErrNilNode
	}
	t.wLock()
	defer t.wUnlock()
	if newEl == nil || refEl == nil {
		return ErrNilNode
	}
	idx := t.findNodeIndex(refEl)
	if idx == -1 {
		return ErrNotFound
	}
	t.adoptChild(newEl)
	t.childs = append(t.childs[:idx], append([]Node{newEl}, t.childs[idx:]...)...)

	if t.elementmap == nil {
		t.elementmap = make(map[string][]*Element)
	}
	t.elementmap[newEl.Name] = append(t.elementmap[newEl.Name], newEl)

	// 更新计数
	atomic.AddInt64(&t.childCount, 1)
	p := t
	for p != nil {
		atomic.AddInt64(&p.descCount, newEl.descCount)
		p = p.parent
	}
	return nil
}

// InsertAfter 在引用节点后插入
func (t *Element) InsertAfter(newEl, refEl *Element) error {
	if t == nil {
		return ErrNilNode
	}
	t.wLock()
	defer t.wUnlock()
	if newEl == nil || refEl == nil {
		return ErrNilNode
	}
	idx := t.findNodeIndex(refEl)
	if idx == -1 {
		return ErrNotFound
	}
	insertPos := idx + 1
	t.adoptChild(newEl)
	t.childs = append(t.childs[:insertPos], append([]Node{newEl}, t.childs[insertPos:]...)...)

	if t.elementmap == nil {
		t.elementmap = make(map[string][]*Element)
	}
	t.elementmap[newEl.Name] = append(t.elementmap[newEl.Name], newEl)

	atomic.AddInt64(&t.childCount, 1)
	p := t
	for p != nil {
		atomic.AddInt64(&p.descCount, newEl.descCount)
		p = p.parent
	}
	return nil
}

// ReplaceChild 替换子节点
func (t *Element) ReplaceChild(oldEl, newEl *Element) error {
	if t == nil {
		return ErrNilNode
	}
	t.wLock()
	defer t.wUnlock()
	if oldEl == nil || newEl == nil {
		return ErrNilNode
	}
	idx := t.findNodeIndex(oldEl)
	if idx == -1 {
		return ErrNotFound
	}

	// 移除旧节点
	t.removeFromMap(oldEl)
	oldEl.parent = nil
	oldEl.syncRoot(oldEl)

	atomic.AddInt64(&t.childCount, -1)
	p := t
	for p != nil {
		atomic.AddInt64(&p.descCount, -oldEl.descCount)
		p = p.parent
	}

	// 接入新节点
	t.adoptChild(newEl)
	t.childs[idx] = newEl

	if t.elementmap == nil {
		t.elementmap = make(map[string][]*Element)
	}
	t.elementmap[newEl.Name] = append(t.elementmap[newEl.Name], newEl)

	atomic.AddInt64(&t.childCount, 1)
	p = t
	for p != nil {
		atomic.AddInt64(&p.descCount, newEl.descCount)
		p = p.parent
	}
	return nil
}

// RemoveChild 删除指定元素
func (t *Element) RemoveChild(el *Element) bool {
	return removeNodeFromParent(el)
}

// RemoveNodeByName 删除所有同名直接子元素
func (t *Element) RemoveNodeByName(name string) bool {
	if t == nil || t.elementmap == nil {
		return false
	}
	t.wLock()
	defer t.wUnlock()

	if _, ok := t.elementmap[name]; !ok {
		return false
	}
	delete(t.elementmap, name)

	var newChilds []Node
	removedCount := int64(0)
	removedDesc := int64(0)
	for _, v := range t.childs {
		if el, ok := v.(*Element); ok && el.Name == name {
			el.parent = nil
			el.syncRoot(el)
			removedCount++
			removedDesc += el.descCount
			continue
		}
		newChilds = append(newChilds, v)
	}
	t.childs = newChilds

	atomic.AddInt64(&t.childCount, -removedCount)
	p := t
	for p != nil {
		atomic.AddInt64(&p.descCount, -removedDesc)
		p = p.parent
	}
	return true
}

// RemoveAll 清空所有子节点
func (t *Element) RemoveAll() {
	if t == nil {
		return
	}
	t.wLock()
	defer t.wUnlock()

	// 解除所有引用
	totalDesc := int64(0)
	for _, child := range t.childs {
		if el, ok := child.(*Element); ok {
			el.parent = nil
			el.syncRoot(el)
			totalDesc += el.descCount
		}
		child.setParent(nil)
	}

	t.childs = t.childs[:0]
	t.elementmap = nil
	atomic.StoreInt64(&t.childCount, 0)

	p := t
	for p != nil {
		atomic.AddInt64(&p.descCount, -totalDesc)
		p = p.parent
	}
}

// ===================== 克隆 =====================

// CloneElement 克隆元素节点
func (t *Element) CloneElement(deep bool) *Element {
	if t == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	return t.clone(deep, nil)
}

func (t *Element) clone(deep bool, newRoot *Element) *Element {
	el := &Element{
		Space:      t.Space,
		Prefix:     t.Prefix,
		Name:       t.Name,
		childs:     make([]Node, 0),
		elementmap: nil,
		childCount: 0,
		descCount:  1,
	}

	if newRoot == nil {
		el.lc = new(sync.RWMutex)
		el.root = el
		newRoot = el
	} else {
		el.lc = newRoot.lc
		el.root = newRoot
	}

	// 复制属性
	if len(t.Attrs) > 0 {
		el.Attrs = make([]*Attr, len(t.Attrs))
		copy(el.Attrs, t.Attrs)
		el.attrmap = make(map[string]string, len(t.attrmap))
		for k, v := range t.attrmap {
			el.attrmap[k] = v
		}
	}

	if len(t.spaceNS) > 0 {
		el.spaceNS = make(map[string]string, len(t.spaceNS))
		for k, v := range t.spaceNS {
			el.spaceNS[k] = v
		}
	}

	if deep {
		for _, child := range t.childs {
			switch c := child.(type) {
			case *Element:
				childCloned := c.clone(true, newRoot)
				childCloned.parent = el
				el.appendChild(childCloned)
			case *TextNode:
				el.appendChild(&TextNode{Value: c.Value})
			case *CDATANode:
				el.appendChild(&CDATANode{Value: c.Value})
			case *CommentNode:
				el.appendChild(&CommentNode{Text: c.Text})
			case *ProcInstNode:
				el.appendChild(&ProcInstNode{Target: c.Target, Data: c.Data})
			}
		}
	}
	return el
}

// ===================== 导航增强 =====================

// IsRoot 是否根节点
func (t *Element) IsRoot() bool {
	if t == nil {
		return false
	}
	return t.parent == nil
}

// IsLeaf 是否叶子节点（无子元素）
func (t *Element) IsLeaf() bool {
	if t == nil {
		return true
	}
	t.rLock()
	defer t.rUnlock()
	return atomic.LoadInt64(&t.childCount) == 0
}

// HasChildNodes 是否有子节点
func (t *Element) HasChildNodes() bool {
	if t == nil {
		return false
	}
	t.rLock()
	defer t.rUnlock()
	return len(t.childs) > 0
}

// Depth 当前节点深度（根节点为 1）
func (t *Element) Depth() int {
	if t == nil {
		return 0
	}
	depth := 1
	p := t.parent
	for p != nil {
		depth++
		p = p.parent
	}
	return depth
}

// Ancestors 获取所有祖先节点（从父到根）
func (t *Element) Ancestors() []*Element {
	if t == nil {
		return nil
	}
	var res []*Element
	p := t.parent
	for p != nil {
		res = append(res, p)
		p = p.parent
	}
	return res
}

// Siblings 获取所有兄弟元素
func (t *Element) Siblings() []*Element {
	if t == nil || t.parent == nil {
		return nil
	}
	t.rLock()
	defer t.rUnlock()
	var res []*Element
	for _, child := range t.parent.childs {
		if el, ok := child.(*Element); ok && el != t {
			res = append(res, el)
		}
	}
	return res
}

// NodePath 获取节点路径字符串
func (t *Element) NodePath() string {
	if t == nil {
		return ""
	}
	var parts []string
	cur := t
	for cur != nil {
		parts = append([]string{cur.Name}, parts...)
		cur = cur.parent
	}
	return strings.Join(parts, "/")
}

// ===================== 命名空间工具 =====================

// LookupPrefix 根据命名空间 URI 查找前缀
func (t *Element) LookupPrefix(uri string) string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()

	for _, attr := range t.Attrs {
		if strings.HasPrefix(attr.Name, "xmlns:") && attr.Value == uri {
			return strings.TrimPrefix(attr.Name, "xmlns:")
		}
	}

	if t.parent != nil {
		return t.parent.LookupPrefix(uri)
	}
	return ""
}

// LookupNamespaceURI 根据前缀查找命名空间 URI
func (t *Element) LookupNamespaceURI(prefix string) string {
	if t == nil {
		return ""
	}
	t.rLock()
	defer t.rUnlock()

	if prefix == "" {
		if v, ok := t.AttrValue("xmlns"); ok {
			return v
		}
	} else {
		if v, ok := t.spaceNS[prefix]; ok {
			return v
		}
		if v, ok := t.AttrValue("xmlns:" + prefix); ok {
			return v
		}
	}
	if t.parent != nil {
		return t.parent.LookupNamespaceURI(prefix)
	}
	return ""
}

// ChildrenLength 直接子元素数量（O(1)）
func (t *Element) ChildrenLength() int64 {
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.childCount)
}

// ChildLengthByName 同名直接子元素数量（O(1)）
func (t *Element) ChildLengthByName(name string) int64 {
	if t == nil || t.elementmap == nil {
		return 0
	}
	t.rLock()
	defer t.rUnlock()
	els, ok := t.elementmap[name]
	if !ok {
		return 0
	}
	return int64(len(els))
}

// DocLength 子树元素总数（O(1)）
func (t *Element) DocLength() int64 {
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.descCount)
}

// xmlEscape XML 特殊字符转义
func xmlEscape(s string, isAttr bool) string {
	var b strings.Builder
	b.Grow(len(s) + 16)
	for _, c := range s {
		switch c {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			if isAttr {
				b.WriteString("&quot;")
			} else {
				b.WriteRune(c)
			}
		case '\'':
			if isAttr {
				b.WriteString("&apos;")
			} else {
				b.WriteRune(c)
			}
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
