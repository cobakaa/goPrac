package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// код писать тут

type tpl struct {
	ToGen   map[string][]GenFunc
	Structs map[string][]StructField
	JsonMap map[string]string
}

var funcMap = template.FuncMap{
	// The name "inc" is what the function will be called in the template text.
	"inc": func(i int) int {
		return i + 1
	},
}

var (
	serveTpl = template.Must(template.New("serveTpl").Funcs(funcMap).Parse(`
	func CompStrSlices(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	{{$Strs := .Structs}}
	{{$Jm := .JsonMap}}
	{{range $StructName, $Pair := .ToGen}}
	// {{$StructName}}
	func (h *{{$StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		{{range $Pair}}
		case {{.Url}}:
			h.wrapper{{.Name}}(w, r)
		{{end}}
		default:
			response := Response{}
			w.WriteHeader(http.StatusNotFound)
			response.Err = "unknown method"
			resp, _ := json.Marshal(response)
			fmt.Fprintf(w, string(resp))
		}
	}

	{{$Name := $StructName}}
	{{range $Pair}}
	func (h *{{$Name}}) wrapper{{.Name}} (w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		response := Response{}
		
		{{if .Auth}}
			authKey := []string{"100500"}
			if !CompStrSlices(r.Header["X-Auth"], authKey) {
				w.WriteHeader(http.StatusForbidden)
				response.Err = "unauthorized"
				resp, _ := json.Marshal(response)
				fmt.Fprintf(w, string(resp))
				return
			}
		{{end}}

		{{if ne .Method ""}}
			if r.Method != {{.Method}} {
				response.Err = "bad method"
				resp, _ := json.Marshal(response)
				w.WriteHeader(http.StatusNotAcceptable)
				fmt.Fprintf(w, string(resp))
				return
			}
		{{end}}

		params := {{.ParamType}}{
			{{range index $Strs .ParamType}}
			{{if eq .Type "string"}}{{.FieldName}}: r.FormValue("{{.LowerFieldName}}"), {{end}}
			{{end}}
		}
		{{range index $Strs .ParamType}}
			{{if eq .Type "int"}}
				{{.LowerFieldName}}, errI := strconv.Atoi(r.FormValue("{{.LowerFieldName}}"))
				if errI != nil {
					response.Err = "{{.LowerFieldName}} must be int"
					resp, _ := json.Marshal(response)
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, string(resp))
					return
				}
				params.{{.FieldName}} = {{.LowerFieldName}}
			{{end}}
		{{end}}
		{{range index $Strs .ParamType}}
		{{$Type := .Type}}
		{{$Fn := .FieldName}}
		{{$Lfn := .LowerFieldName}}
		{{$Enum := .Enum}}
		{{$Ep := .EnumParams}}
		{{$Def := .Default}}
		{{$Enumed := false}}
		{{$MinFlag := false}}
		{{$MaxFlag := false}}
		{{$DefF := false}}
		{{$MinF := .MinF}}
		{{$MaxF := .MaxF}}
		{{$Min := .Min}}
		{{$Max := .Max}}
			{{range .Param}}
				{{if eq . "required"}}
					{{if eq $Type "string"}}
						if params.{{$Fn}} == "" {
					{{else}}
						if params.{{$Fn}} == 0 {
					{{end}}
							response.Err = "{{$Lfn}} must me not empty"
							
							resp, _ := json.Marshal(response)
							 w.WriteHeader(http.StatusBadRequest)
							fmt.Fprintf(w, string(resp))
							return
						}
				{{end}}
				{{if and (not $DefF) (ne $Def "")}}
					{{$DefF = true}}
					if params.{{$Fn}} == "" {
						params.{{$Fn}} = "{{$Def}}"
					}
				{{end}}
				{{if and (not $Enumed) $Enum}}
					{{$Enumed = true}}
						if {{range $i, $v := $Ep}}params.{{$Fn}} != "{{$v}}"{{if ne (inc $i) (len $Ep)}} && {{end}}{{end}} {
							response.Err = "{{$Lfn}} must be one of [{{range $i, $v := $Ep}}{{$v}}{{if ne (inc $i) (len $Ep)}}, {{end}}{{end}}]"
							resp, _ := json.Marshal(response)
							w.WriteHeader(http.StatusBadRequest)
							fmt.Fprintf(w, string(resp))
							return
						}
				{{end}}
				{{if $MinF}}
					{{if not $MinFlag}}
					{{$MinFlag = true}}
					{{if eq $Type "int"}}
						if params.{{$Fn}} < {{$Min}} {
							response.Err = "{{$Lfn}} must be >= {{$Min}}"
							resp, _ := json.Marshal(response)
							w.WriteHeader(http.StatusBadRequest)
							fmt.Fprintf(w, string(resp))
							return
						}
					{{else}}
						if len(params.{{$Fn}}) < {{$Min}} {
							response.Err = "{{$Lfn}} len must be >= {{$Min}}"
							resp, _ := json.Marshal(response)
							w.WriteHeader(http.StatusBadRequest)
							fmt.Fprintf(w, string(resp))
							return
						}
					{{end}}
					{{end}}
				{{end}}
				{{if $MaxF}}
				{{if not $MaxFlag}}
					{{$MaxFlag = true}}
					if params.{{$Fn}} > {{$Max}} {
						response.Err = "{{$Lfn}} must be <= {{$Max}}"
						resp, _ := json.Marshal(response)
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprintf(w, string(resp))
						return
					}
				{{end}}
				{{end}}
			{{end}}
		{{end}}
		res, err := h.{{.Name}}(ctx, params)
		if err != nil {
			response.Err = err.Error()
			if reflect.TypeOf(err).Name() == "ApiError" {
				w.WriteHeader(err.(ApiError).HTTPStatus)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			resp, _ := json.Marshal(response)
			//w.WriteHeader(err.(*ApiError).HTTPStatus)
			fmt.Fprintf(w, string(resp))
			return
		}
		response.Resp = res
		response.Err = ""
		resp, _ := json.Marshal(response)
		fmt.Fprintf(w, string(resp))
	}
	{{end}}
	{{end}}
`))
)

func SubstrParam(s string, param string) string {
	first := strings.Index(s, `"`+param+`":`)
	if first == -1 {
		return ""
	}
	i := first
	for s[i] != ',' && s[i] != '}' {
		i++
	}
	return s[first+len(param)+4 : i]
}

type GenFunc struct {
	Url       string
	Name      string
	ParamType string
	Method    string
	Auth      bool
}

type StructField struct {
	FieldName      string
	Type           string
	Param          []string
	EnumParams     []string
	Enum           bool
	LowerFieldName string
	Default        string
	Min            int
	MinF           bool
	Max            int
	MaxF           bool
}

type ApiError struct {
	HTTPStatus int
	Err        error
}

type Response struct {
	Err  interface{}
	Resp interface{}
}

//type GenStruct struct {
//	Name   string
//	Params []Field
//}

func main() {

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "fmt"`)
	fmt.Fprintln(out, `import "reflect"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out)

	fmt.Fprintln(out, `
		type Response struct {
			Err  string `+"`json:\"error\"`\n"+
		`Resp interface{}`+"`json:\"response,omitempty\"`\n"+
		`}
	`)

	toGen := map[string][]GenFunc{}
	structToGen := map[string][]StructField{}
	JsonMapping := map[string]string{}

	for _, f := range node.Decls {
		g, ok := f.(*ast.FuncDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.FuncDecls\n", f)
			continue
		}

		if g.Doc == nil {
			fmt.Printf("SKIP func %#v doesnt have comments\n", g.Name.Name)
			continue
		}

		needCodegen := false
		for _, comment := range g.Doc.List {
			needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
		}

		if !needCodegen {
			fmt.Printf("SKIP func %#v doesnt have apigen mark\n", g.Name.Name)
		}

		fmt.Printf("process func %#v\n", g.Name.Name)
		structName := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		//fmt.Printf("\t generating ServeHTTP method for %s\n", structName)
		for _, comment := range g.Doc.List {
			url := SubstrParam(comment.Text, "url")
			method := SubstrParam(comment.Text, "method")
			auth, _ := strconv.ParseBool(SubstrParam(comment.Text, "auth"))
			toGen[structName] = append(toGen[structName], GenFunc{url, g.Name.Name,
				g.Type.Params.List[1].Type.(*ast.Ident).Name, method, auth})
		}

	}

	fmt.Printf("LOOP for Structs\n")
	for _, f := range node.Decls {
		g, ok := f.(*ast.GenDecl)
		if !ok {
			fmt.Printf("\tSKIP %T is not *ast.GenDecls\n", f)
			continue
		}

		//SPECS_LOOP:
		for _, spec := range g.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				fmt.Printf("\tSKIP %T is not ast.TypeSpec\n", currType)
				continue
			}

			currStruct, ok := currType.Type.(*ast.StructType)
			if !ok {
				fmt.Printf("\tSKIP %T is not ast.StructType\n")
				continue
			}

			for _, tag := range currStruct.Fields.List {
				if tag.Tag == nil {
					continue
				}
				//haveTag := false
				tagVal := tag.Tag.Value
				substrApi := "apivalidator:"
				indApi := strings.Index(tagVal, substrApi)
				if indApi != -1 {
					paramApi := tagVal[indApi+len(substrApi)+1 : len(tagVal)-1-1]
					params := strings.Split(paramApi, ",")
					lowerName := strings.ToLower(tag.Names[0].Name)
					var enumParam []string
					enumF := false
					defParam := ""
					min := 0
					max := 0
					MinF := false
					MaxF := false
					for _, p := range params {
						str := "paramname="
						indName := strings.Index(p, str)
						if indName != -1 {
							lowerName = p[indName+len(str):]
						}

						enum := "enum="
						indEnum := strings.Index(p, enum)
						if indEnum != -1 {
							enumF = true
							enumParam = strings.Split(p[indEnum+len(enum):], "|")
						}

						def := "default="
						defParam = ""
						indDef := strings.Index(p, def)
						if indDef != -1 {
							defParam = p[indDef+len(def):]
						}

						minStr := "min="
						indMin := strings.Index(p, minStr)
						if indMin != -1 {
							MinF = true
							min, _ = strconv.Atoi(p[indMin+len(minStr):])
						}

						maxStr := "max="
						indMax := strings.Index(p, maxStr)
						if indMax != -1 {
							MaxF = true
							max, _ = strconv.Atoi(p[indMax+len(minStr):])
						}
					}
					structToGen[currType.Name.Name] = append(structToGen[currType.Name.Name],
						StructField{tag.Names[0].Name, tag.Type.(*ast.Ident).Name, params,
							enumParam, enumF, lowerName,
							defParam, min, MinF, max, MaxF})
				} else {
					substrJson := "json:"
					indJson := strings.Index(tagVal, substrJson)
					if indJson == -1 {
						continue
					}

					paramJson := tagVal[indJson+len(substrJson)+1 : len(tagVal)-1-1]
					JsonMapping[tag.Names[0].Name] = paramJson
				}

			}

			//fmt.Printf("type: %T data: %+v\n", structToGen, structToGen)
		}

	}

	fmt.Printf("\t generating Template\n")
	serveTpl.Execute(out, tpl{toGen, structToGen, JsonMapping})

}
