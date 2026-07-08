// Copyright (c) 2016, donnie <donnie4w@gmail.com>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// github.com/donnie4w/dom4g

package dom4g

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func LoadByXmlExtend(xmlstr string) (*Element, error) {
	return LoadByStreamExtend(strings.NewReader(xmlstr))
}

const __CDATA_ = "__CDATA_PKJGNSZHWH_"

var cdataRegex = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)]]>`)

func LoadByStreamExtend(r io.Reader) (*Element, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	content := string(buf)

	var cdataStore []string
	content = cdataRegex.ReplaceAllStringFunc(content, func(raw string) string {
		match := cdataRegex.FindStringSubmatch(raw)
		if len(match) < 2 {
			return raw
		}
		inner := match[1]
		cdataStore = append(cdataStore, inner)
		idx := len(cdataStore) - 1

		var sb strings.Builder
		sb.WriteString(__CDATA_)
		sb.WriteString(strconv.Itoa(idx))
		sb.WriteString(`__`)
		return sb.String()
	})

	return loadByStream(bytes.NewBufferString(content), cdataStore, defaultParseOption)
}

type textSegment struct {
	IsPlaceholder bool   // 是否CDATA占位符
	Index         string // 占位符下标
	Text          string // 普通文本内容
}

// splitCDATAPlaceholder 扫描字符串，切割出普通文本与CDATA占位符
func splitCDATAPlaceholder(s string) []textSegment {
	const prefix = __CDATA_
	const suffix = `__`
	var res []textSegment
	n := len(s)
	i := 0

	for i < n {
		// 匹配占位符开头
		if i+len(prefix) <= n && s[i:i+len(prefix)] == prefix {
			j := i + len(prefix)
			// 向后找后缀 __
			found := false
			for j <= n-len(suffix) {
				if s[j:j+len(suffix)] == suffix {
					// 提取中间数字
					idxStr := s[i+len(prefix) : j]
					res = append(res, textSegment{
						IsPlaceholder: true,
						Index:         idxStr,
					})
					i = j + len(suffix)
					found = true
					break
				}
				j++
			}
			if !found {
				// 找不到闭合后缀，当作普通文本
				res = append(res, textSegment{
					IsPlaceholder: false,
					Text:          string(s[i]),
				})
				i++
			}
		} else {
			start := i
			for i < n && !(i+len(prefix) <= n && s[i:i+len(prefix)] == prefix) {
				i++
			}
			res = append(res, textSegment{
				IsPlaceholder: false,
				Text:          s[start:i],
			})
		}
	}
	return res
}
