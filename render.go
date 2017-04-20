// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tpl

import (
	"bytes"
	"errors"
	"html/template"
	"io"
)

type render struct {
	TplName        string
	Layout         string
	LayoutSections map[string]string
	Data           map[string]interface{}
}

func (r *render) Render(w io.Writer) error {
	b, e := r.RenderBytes()
	if e != nil {
		return e
	}
	_, e = w.Write(b)
	return e
}

func (r *render) RenderString() (string, error) {
	b, e := r.RenderBytes()
	return string(b), e
}

// RenderBytes returns the bytes of rendered template string. Do not send out response.
func (r *render) RenderBytes() ([]byte, error) {
	buf, err := r.renderTemplate()
	//if the controller has set layout, then first get the tplName's content set the content to the layout
	if err == nil && r.Layout != "" {
		r.Data["LayoutContent"] = template.HTML(buf.String())
		if r.LayoutSections != nil {
			for sectionName, sectionTpl := range r.LayoutSections {
				if sectionTpl == "" {
					r.Data[sectionName] = ""
					continue
				}
				buf.Reset()
				err = executeTemplate(&buf, sectionTpl, r.Data)
				if err != nil {
					return nil, err
				}
				r.Data[sectionName] = template.HTML(buf.String())
			}
		}
		buf.Reset()
		executeTemplate(&buf, r.Layout, r.Data)
	}
	return buf.Bytes(), err
}

func (r *render) renderTemplate() (bytes.Buffer, error) {
	var buf bytes.Buffer
	if r.TplName == "" {
		return buf, errors.New("tplname is null")
	}
	if runmode == DEV {
		buildFiles := []string{r.TplName}
		if r.Layout != "" {
			buildFiles = append(buildFiles, r.Layout)
			if r.LayoutSections != nil {
				for _, sectionTpl := range r.LayoutSections {
					if sectionTpl == "" {
						continue
					}
					buildFiles = append(buildFiles, sectionTpl)
				}
			}
		}
		BuildTemplate(viewsPath, buildFiles...)
	}
	return buf, executeTemplate(&buf, r.TplName, r.Data)
}

func NewRender() *render {
	return &render{
		Data: make(map[string]interface{}),
	}
}
