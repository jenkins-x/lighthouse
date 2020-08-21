package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

func getTagValue(tag *ast.BasicLit, expression string) string {
	if tag == nil {
		return ""
	}
	r := regexp.MustCompile(expression)
	result := r.FindStringSubmatch(tag.Value)
	if len(result) > 1 {
		return result[1]
	}
	return ""
}

type _type struct {
	Name         string
	Package      string
	IsPointer    bool
	ArrayEltType *_type
	MapKeyType   *_type
	MapValueType *_type
}

func (t *_type) Format(structs map[string]_struct) string {
	if t.ArrayEltType != nil {
		return fmt.Sprintf("[]%s", t.ArrayEltType.Format(structs))
	}
	if t.MapKeyType != nil && t.MapValueType != nil {
		return fmt.Sprintf("map[%s]%s", t.MapKeyType.Format(structs), t.MapValueType.Format(structs))
	}
	result := t.Name
	if t.Package != "" {
		result = fmt.Sprintf("%s.%s", t.Package, result)
	}
	if _, ok := structs[result]; ok {
		result = fmt.Sprintf("[%s](#%s)", result, result)
	}
	if t.IsPointer {
		result = fmt.Sprintf("*%s", result)
	}
	return result
}

type _field struct {
	Name     string
	Stanza   string
	Desc     string
	Type     *_type
	Required bool
}

type _struct struct {
	Desc   string
	Fields []_field
}

func parseType(typeExpr ast.Expr) *_type {
	switch t := typeExpr.(type) {
	case *ast.Ident:
		return &_type{
			Name: t.Name,
		}
	case *ast.ArrayType:
		return &_type{
			ArrayEltType: parseType(t.Elt),
		}
	case *ast.MapType:
		return &_type{
			MapKeyType:   parseType(t.Key),
			MapValueType: parseType(t.Value),
		}
	case *ast.SelectorExpr:
		x := parseType(t.Sel)
		x.Package = t.X.(*ast.Ident).Name
		return x
	case *ast.StarExpr:
		x := parseType(t.X)
		x.IsPointer = true
		return x
	default:
		return nil
	}
}

func parseField(name string, field *ast.Field) _field {
	f := _field{
		Name:   name,
		Stanza: getTagValue(field.Tag, `json:\"([^,\"]*).*\"`),
		Desc:   strings.ReplaceAll(strings.TrimSpace(field.Doc.Text()), "\n", "<br />"),
		Type:   parseType(field.Type),
	}
	if !(f.Type.ArrayEltType != nil || f.Type.MapKeyType != nil || f.Type.MapValueType != nil || f.Type.IsPointer) {
		f.Required = !strings.Contains(getTagValue(field.Tag, `json:\"(.*)\"`), "omitempty")
	}
	return f
}

func parseStruct(typeSpec *ast.TypeSpec, doc *ast.CommentGroup) _struct {
	s := _struct{
		Desc: strings.ReplaceAll(strings.TrimSpace(doc.Text()), "\n", "<br />"),
	}

	structure, ok := typeSpec.Type.(*ast.StructType)
	if ok {
		for _, field := range structure.Fields.List {
			name := ""
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			s.Fields = append(s.Fields, parseField(name, field))
		}
	}

	return s
}

const tmpl = `# {{ .Title }}

{{ range $name, $struct := .Structs -}}
- [{{ $name }}](#{{ $name }})
{{ end }}

{{ range $name, $struct := .Structs -}}
## {{ $name }}

{{ if $struct.Desc }}{{ $struct.Desc }}{{ end }}

{{ if $struct.Fields -}}
| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
{{- range $struct.Fields }}
| {{ .Name }} | {{ if .Stanza }}` + "`" + `{{ .Stanza }}` + "`" + `{{ end }} | {{ .Type.Format $.Structs }} | {{ if .Required }}Yes{{ else }}No{{ end }} | {{ .Desc }} |
{{- end }}
{{- end }}

{{ end }}
`

var inputFiles = pflag.StringArray("input-file", []string{}, "input go file to parse")
var outputPath = pflag.String("output-path", "", "output path")
var title = pflag.String("title", "", "doc title")
var help = pflag.Bool("help", false, "Prints defaults")

func main() {
	pflag.Parse()

	if *help {
		pflag.PrintDefaults()
		return
	}

	if inputFiles == nil || len(*inputFiles) == 0 {
		fmt.Println("You must specify at least one input file.")
		pflag.PrintDefaults()
		return
	}
	if outputPath == nil || *outputPath == "" {
		fmt.Println("You must specify an output path.")
		pflag.PrintDefaults()
		return
	}
	if title == nil || *title == "" {
		fmt.Println("You must provide a title for your doc.")
		pflag.PrintDefaults()
		return
	}

	t := template.New("struct-docs")
	if _, err := t.Parse(tmpl); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(*outputPath, os.ModePerm); err != nil {
		panic(err)
	}

	structs := map[string]_struct{}

	fileSet := token.NewFileSet()
	for _, file := range *inputFiles {
		node, _ := parser.ParseFile(fileSet, file, nil, parser.ParseComments)

		for _, decl := range node.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if ok {
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.IsExported() {
						structs[typeSpec.Name.Name] = parseStruct(typeSpec, genDecl.Doc)
					}
				}
			}
		}
	}

	f, err := os.Create(path.Join(*outputPath, *title+".md"))
	if err != nil {
		panic(err)
	}

	if err := t.Execute(f, map[string]interface{}{
		"Title":   *title,
		"Structs": structs,
	}); err != nil {
		panic(err)
	}
}
