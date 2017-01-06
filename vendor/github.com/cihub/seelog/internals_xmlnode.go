// Copyright (c) 2012 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

type xmlNode struct {
	name       string
	attributes map[string]string
	children   []*xmlNode
	value      string
}

func newNode() *xmlNode {
	node := new(xmlNode)
	node.children = make([]*xmlNode, 0)
	node.attributes = make(map[string]string)
	return node
}

func (node *xmlNode) String() string {
	str := fmt.Sprintf("<%s", node.name)

	for attrName, attrVal := range node.attributes {
		str += fmt.Sprintf(" %s=\"%s\"", attrName, attrVal)
	}

	str += ">"
	str += node.value

	if len(node.children) != 0 {
		for _, child := range node.children {
			str += fmt.Sprintf("%s", child)
		}
	}

	str += fmt.Sprintf("</%s>", node.name)

	return str
}

func (node *xmlNode) unmarshal(startEl xml.StartElement) error {
	node.name = startEl.Name.Local

	for _, v := range startEl.Attr {
		_, alreadyExists := node.attributes[v.Name.Local]
		if alreadyExists {
			return errors.New("tag '" + node.name + "' has duplicated attribute: '" + v.Name.Local + "'")
		}
		node.attributes[v.Name.Local] = v.Value
	}

	return nil
}

func (node *xmlNode) add(child *xmlNode) {
	if node.children == nil {
		node.children = make([]*xmlNode, 0)
	}

	node.children = append(node.children, child)
}

func (node *xmlNode) hasChildren() bool {
	return node.children != nil && len(node.children) > 0
}

//=============================================

func unmarshalConfig(reader io.Reader) (*xmlNode, error) {
	xmlParser := xml.NewDecoder(reader)

	config, err := unmarshalNode(xmlParser, nil)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("xml has no content")
	}

	nextConfigEntry, err := unmarshalNode(xmlParser, nil)
	if nextConfigEntry != nil {
		return nil, errors.New("xml contains more than one root element")
	}

	return config, nil
}

func unmarshalNode(xmlParser *xml.Decoder, curToken xml.Token) (node *xmlNode, err error) {
	firstLoop := true
	for {
		var tok xml.Token
		if firstLoop && curToken != nil {
			tok = curToken
			firstLoop = false
		} else {
			tok, err = getNextToken(xmlParser)
			if err != nil || tok == nil {
				return
			}
		}

		switch tt := tok.(type) {
		case xml.SyntaxError:
			err = errors.New(tt.Error())
			return
		case xml.CharData:
			value := strings.TrimSpace(string([]byte(tt)))
			if node != nil {
				node.value += value
			}
		case xml.StartElement:
			if node == nil {
				node = newNode()
				err := node.unmarshal(tt)
				if err != nil {
					return nil, err
				}
			} else {
				childNode, childErr := unmarshalNode(xmlParser, tok)
				if childErr != nil {
					return nil, childErr
				}

				if childNode != nil {
					node.add(childNode)
				} else {
					return
				}
			}
		case xml.EndElement:
			return
		}
	}
}

func getNextToken(xmlParser *xml.Decoder) (tok xml.Token, err error) {
	if tok, err = xmlParser.Token(); err != nil {
		if err == io.EOF {
			err = nil
			return
		}
		return
	}

	return
}
