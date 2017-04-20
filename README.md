# tpl
golang template engine  
从[beego](https://beego.me)框架中提取出的模板引擎  
Usage:  
```go get -u github.com/x-croz/tpl```
```go
package main

import (
	"github.com/x-croz/tpl"
	"log"
	"net/http"
)

func init() {
    tpl.SetViewsPath("tpl")// default is "tpl" if not set
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render := tpl.NewRender()
		render.TplName = "index.tpl"
		render.Data["Name"] = "CROZ"
		e := render.Render(w)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		
	})
	log.Fatal(http.ListenAndServe(":2333", nil))
}
```
tpl/index.tpl
```html
<html>
<head>
	<title></title>
</head>
<body>
	{{template "header.tpl" .}}
	Welcome , {{.Name}}!
</body>
</html>
```
tpl/header.tpl
```html
<p>this is header</p>
```
