# tpl
golang template engine

从[beego](https://beego.me)框架中提取出的模板引擎

# usage

```goget -u github.com/singsenxc/tpl```
```
import (
	"github.com/singsenxc/tpl"
	"log"
	"net/http"
)

func main() {
	tpl.SetViewsPath("views")	//if not set, default is "views"
	tpl.SetTemplateLeft("{{")	//deflult is "{{"
	tpl.SetTemplateRight("}}")	//deflult is "}}"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render := tpl.Render{Data: make(map[string]interface{})}
		render.TplName = "index.tpl"
		render.Data["Name"] = "Singsen"
		b, e := render.RenderBytes()
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	log.Fatal(http.ListenAndServe(":2333", nil))
}
```
views/index.tpl
```
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
views/header.tpl
```
<p>this is header</p>
```
