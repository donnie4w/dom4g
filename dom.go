package dom4g

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type E interface {
	ToString() string
}

type Attr struct {
	Name  string
	Value string
}

type Element struct {
	Name       string
	Value      string
	Attrs      []*Attr
	childs     []E
	parent     E
	elementmap map[string][]E
	attrmap    map[string]string
	lc         *sync.RWMutex
}

func LoadByXml(xmlstr string) (current *Element, err error) {
	defer func() {
		if er := recover(); er != nil {
			fmt.Println(er)
			err = errors.New("xml load error!")
		}
	}()
	s := strings.NewReader(xmlstr)
	decoder := xml.NewDecoder(s)
	isRoot := true
	for t, er := decoder.Token(); er == nil; t, er = decoder.Token() {
		switch token := t.(type) {
		case xml.StartElement:
			el := new(Element)
			el.Name = token.Name.Local
			el.Attrs = make([]*Attr, 0)
			el.childs = make([]E, 0)
			el.elementmap = make(map[string][]E, 0)
			el.attrmap = make(map[string]string, 0)
			el.lc = new(sync.RWMutex)
			for _, a := range token.Attr {
				ar := new(Attr)
				ar.Name = a.Name.Local
				ar.Value = a.Value
				el.Attrs = append(el.Attrs, ar)
				el.attrmap[ar.Name] = ar.Value
			}
			if isRoot {
				isRoot = false
			} else {
				current.childs = append(current.childs, el)
				current.elementmap[el.Name] = append(current.elementmap[el.Name], el)
				el.parent = current
			}
			current = el
		case xml.EndElement:
			if current.parent != nil {
				current = current.parent.(*Element)
			}
		case xml.CharData:
			current.Value = string([]byte(token))
		default:
			panic("parse xml fail!")
		}
	}
	return current, nil
}

func (t *Element) ToString() string {
	s := fmt.Sprint("<", t.Name)
	sattr := ""
	if len(t.Attrs) > 0 {
		for _, att := range t.Attrs {
			sattr = fmt.Sprint(sattr, " ", att.Name, "=", "\"", att.Value, "\"")
		}
	}
	s = fmt.Sprint(s, sattr, ">")
	if len(t.childs) > 0 {
		for _, v := range t.childs {
			el := v.(*Element)
			s = fmt.Sprint(s, el.ToString())
		}
		return fmt.Sprint(s, t.Value, "</", t.Name, ">")
	} else {
		return toStr(t)
	}
}

func toStr(t *Element) string {
	sattr := ""
	if len(t.Attrs) > 0 {
		for _, att := range t.Attrs {
			sattr = fmt.Sprint(sattr, " ", att.Name, "=", "\"", att.Value, "\"")
		}
	}
	return fmt.Sprint("<", t.Name, sattr, ">", t.Value, "</", t.Name, ">")
}

func (t *Element) Node(name string) *Element {
	t.lc.RLock()
	defer t.lc.RUnlock()
	es, ok := t.elementmap[name]
	if ok {
		el := es[0]
		return el.(*Element)
	} else {
		return nil
	}
}

func (t *Element) Nodes(name string) []*Element {
	t.lc.RLock()
	defer t.lc.RUnlock()
	es, ok := t.elementmap[name]
	if ok {
		ret := make([]*Element, len(es))
		for i, v := range es {
			ret[i] = v.(*Element)
		}
		return ret
	} else {
		return nil
	}
}

func (t *Element) AttrValue(name string) (string, bool) {
	t.lc.RLock()
	defer t.lc.RUnlock()
	v, ok := t.attrmap[name]
	if ok {
		return v, true
	} else {
		return "", false
	}
}

func (t *Element) AddAttr(name, value string) {
	t.lc.Lock()
	defer t.lc.Unlock()
	t.attrmap[name] = value
	isExist := false
	for _, v := range t.Attrs {
		if v.Name == name {
			v.Value = value
			isExist = true
		}
	}
	if !isExist {
		t.Attrs = append(t.Attrs, &Attr{name, value})
	}
}

func (t *Element) RemoveAttr(name string) bool {
	t.lc.Lock()
	defer t.lc.Unlock()
	_, ok := t.attrmap[name]
	if ok {
		delete(t.attrmap, name)
		newAs := make([]*Attr, 0)
		for _, v := range t.Attrs {
			if v.Name != name {
				newAs = append(newAs, v)
			}
		}
		t.Attrs = newAs
		return true
	} else {
		return false
	}
}

func (t *Element) AllNodes() []*Element {
	t.lc.RLock()
	defer t.lc.RUnlock()
	es := t.childs
	if len(es) > 0 {
		ret := make([]*Element, len(es))
		for i, v := range es {
			ret[i] = v.(*Element)
		}
		return ret
	} else {
		return nil
	}
}

func (t *Element) RemoveNode(name string) bool {
	t.lc.Lock()
	defer t.lc.Unlock()
	_, ok := t.elementmap[name]
	if ok {
		delete(t.elementmap, name)
		newCs := make([]E, 0)
		for _, v := range t.childs {
			if v.(*Element).Name != name {
				newCs = append(newCs, v)
			}
		}
		t.childs = newCs
		return true
	} else {
		return false
	}
}

func (t *Element) AddNode(el *Element) error {
	if el.Name == "" {
		return errors.New("error!|name is empty!")
	}
	t.childs = append(t.childs, el)
	el.parent = t
	t.elementmap[el.Name] = append(t.elementmap[el.Name], el)
	return nil
}

func (t *Element) AddNodeByString(xmlstr string) error {
	t.lc.Lock()
	defer t.lc.Unlock()
	el, err := LoadByXml(xmlstr)
	if err != nil {
		return err
	}
	t.AddNode(el)
	return nil
}
