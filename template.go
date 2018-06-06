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
	"errors"
	"fmt"
	"github.com/cr0z/tpl/utils"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	DEV = 0
	PRO = 1
)

var (
	tplFuncMap      = make(template.FuncMap)
	templates       = make(map[string]*template.Template)
	templatesLock   sync.RWMutex
	templateExt     = []string{"html", "tpl"}
	templateEngines = map[string]templatePreProcessor{}
	templateLeft    = "{{"
	templateRight   = "}}"
	runmode         = 0
	viewsPath       = "tpl"
)

func SetRunnmode(mode int) {
	runmode = mode
}

//SetViewsPath default is "views"
func SetViewsPath(path string) {
	viewsPath = path
}

//SetTemplateLeft default is "{{"
func SetTemplateLeft(left string) {
	templateLeft = left
}

//SetTemplateRight default is "}}"
func SetTemplateRight(right string) {
	templateRight = right
}

type templatePreProcessor func(root, path string, funcs template.FuncMap) (*template.Template, error)

type templateFile struct {
	root  string
	files map[string][]string
}

// visit will make the paths into two part,the first is subDir (without tf.root),the second is full path(without tf.root).
// if tf.root="views" and
// paths is "views/errors/404.html",the subDir will be "errors",the file will be "errors/404.html"
// paths is "views/admin/errors/404.html",the subDir will be "admin/errors",the file will be "admin/errors/404.html"
func (tf *templateFile) visit(paths string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() || (f.Mode()&os.ModeSymlink) > 0 {
		return nil
	}
	if !HasTemplateExt(paths) {
		return nil
	}

	replace := strings.NewReplacer("\\", "/")
	file := strings.TrimLeft(replace.Replace(paths[len(tf.root):]), "/")
	subDir := filepath.Dir(file)

	tf.files[subDir] = append(tf.files[subDir], file)
	return nil
}

// HasTemplateExt return this path contains supported template extension of beego or not.
func HasTemplateExt(paths string) bool {
	for _, v := range templateExt {
		if strings.HasSuffix(paths, "."+v) {
			return true
		}
	}
	return false
}

// AddTemplateExt add new extension for template.
func AddTemplateExt(ext string) {
	for _, v := range templateExt {
		if v == ext {
			return
		}
	}
	templateExt = append(templateExt, ext)
}

// BuildTemplate will build all template files in a directory.
// it makes beego can render any template file in view directory.
func BuildTemplate(dir string, files ...string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.New("dir open err")
	}
	self := &templateFile{
		root:  dir,
		files: make(map[string][]string),
	}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		return self.visit(path, f, err)
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return err
	}
	buildAllFiles := len(files) == 0
	for _, v := range self.files {
		for _, file := range v {
			if buildAllFiles || utils.InSlice(file, files) {
				templatesLock.Lock()
				ext := filepath.Ext(file)
				var t *template.Template
				if len(ext) == 0 {
					t, err = getTemplate(self.root, file, v...)
				} else if fn, ok := templateEngines[ext[1:]]; ok {
					t, err = fn(self.root, file, tplFuncMap)
				} else {
					t, err = getTemplate(self.root, file, v...)
				}
				if err != nil {
					// logs.Trace("parse template err:", file, err)
					log.Println("parse template err:", file, err)
				} else {
					templates[file] = t
				}
				templatesLock.Unlock()
			}
		}
	}
	return nil
}

func getTplDeep(root, file, parent string, t *template.Template) (*template.Template, [][]string, error) {
	var fileAbsPath string
	if filepath.HasPrefix(file, "../") {
		fileAbsPath = filepath.Join(root, filepath.Dir(parent), file)
	} else {
		fileAbsPath = filepath.Join(root, file)
	}
	if e := utils.FileExists(fileAbsPath); !e {
		// panic("can't find template file:" + file)
		log.Println("can't find template file:" + file)
		return nil, [][]string{}, errors.New("can't find template file:" + file)
	}
	data, err := ioutil.ReadFile(fileAbsPath)
	if err != nil {
		return nil, [][]string{}, err
	}
	t, err = t.New(file).Parse(string(data))
	if err != nil {
		return nil, [][]string{}, err
	}
	reg := regexp.MustCompile(templateLeft + "[ ]*template[ ]+\"([^\"]+)\"")
	allSub := reg.FindAllStringSubmatch(string(data), -1)
	for _, m := range allSub {
		if len(m) == 2 {
			tl := t.Lookup(m[1])
			if tl != nil {
				continue
			}
			if !HasTemplateExt(m[1]) {
				continue
			}
			t, _, err = getTplDeep(root, m[1], file, t)
			if err != nil {
				return nil, [][]string{}, err
			}
		}
	}
	return t, allSub, nil
}
func getTemplate(root, file string, others ...string) (t *template.Template, err error) {
	t = template.New(file).Delims(templateLeft, templateRight).Funcs(tplFuncMap)
	var subMods [][]string
	t, subMods, err = getTplDeep(root, file, "", t)
	if err != nil {
		return nil, err
	}
	t, err = _getTemplate(t, root, subMods, others...)

	if err != nil {
		return nil, err
	}
	return
}

func _getTemplate(t0 *template.Template, root string, subMods [][]string, others ...string) (t *template.Template, err error) {
	t = t0
	for _, m := range subMods {
		if len(m) == 2 {
			tpl := t.Lookup(m[1])
			if tpl != nil {
				continue
			}
			//first check filename
			for _, otherFile := range others {
				if otherFile == m[1] {
					var subMods1 [][]string
					t, subMods1, err = getTplDeep(root, otherFile, "", t)
					if err != nil {
						// logs.Trace("template parse file err:", err)
						log.Println("template parse file err:", err)
					} else if subMods1 != nil && len(subMods1) > 0 {
						t, err = _getTemplate(t, root, subMods1, others...)
					}
					break
				}
			}
			//second check define
			for _, otherFile := range others {
				fileAbsPath := filepath.Join(root, otherFile)
				data, err := ioutil.ReadFile(fileAbsPath)
				if err != nil {
					continue
				}
				reg := regexp.MustCompile(templateLeft + "[ ]*define[ ]+\"([^\"]+)\"")
				allSub := reg.FindAllStringSubmatch(string(data), -1)
				for _, sub := range allSub {
					if len(sub) == 2 && sub[1] == m[1] {
						var subMods1 [][]string
						t, subMods1, err = getTplDeep(root, otherFile, "", t)
						if err != nil {
							// logs.Trace("template parse file err:", err)
							log.Println("template parse file err:", err)
						} else if subMods1 != nil && len(subMods1) > 0 {
							t, err = _getTemplate(t, root, subMods1, others...)
						}
						break
					}
				}
			}
		}
	}
	return
}

func AddFuncMap(key string, fn interface{}) {
	tplFuncMap[key] = fn
}

func AddTemplateEngine(extension string, fn templatePreProcessor) {
	AddTemplateExt(extension)
	templateEngines[extension] = fn
}

func executeTemplate(wr io.Writer, name string, data interface{}) error {
	if runmode == DEV {
		templatesLock.RLock()
		defer templatesLock.RUnlock()
	}
	if t, ok := templates[name]; ok {
		err := t.ExecuteTemplate(wr, name, data)
		if err != nil {
			log.Println("template Execute err:", err)
		}
		return err
	}
	// panic("can't find templatefile in the path:" + name)
	return errors.New("can't find templatefile in the path:" + name)
}
