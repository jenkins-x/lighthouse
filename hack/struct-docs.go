package main

import (
	"fmt"
	"go/ast"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
	"golang.org/x/tools/go/packages"
)

var t regexp.Regexp

var inputs = pflag.StringArray("input", []string{}, "input to parse")
var roots = pflag.StringArray("root", []string{}, "root structs")
var outputPath = pflag.String("output", "", "output path")
var help = pflag.Bool("help", false, "Prints defaults")

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
	PackageID    string
	IsPointer    bool
	ArrayEltType *_type
	MapKeyType   *_type
	MapValueType *_type
}

func (t *_type) Format() string {
	if t.ArrayEltType != nil {
		return fmt.Sprintf("[]%s", t.ArrayEltType.Format())
	}
	if t.MapKeyType != nil && t.MapValueType != nil {
		return fmt.Sprintf("map[%s]%s", t.MapKeyType.Format(), t.MapValueType.Format())
	}
	result := t.Name
	if t.PackageID != "" {
		result = fmt.Sprintf("[%s](./%s#%s)", result, packageToFile(t.PackageID), result)
	}
	if t.IsPointer {
		result = fmt.Sprintf("*%s", result)
	}
	return result
}

type _field struct {
	Name       string
	IsExported bool
	Stanza     string
	Desc       string
	Type       *_type
	Required   bool
}

type _struct struct {
	Desc       string
	IsExported bool
	Fields     []*_field
}

func (s _struct) FlatFields(structs map[string]map[string]_struct) []*_field {
	ret := []*_field{}
	for _, f := range s.Fields {
		if f.Name != "" {
			if f.IsExported && f.Stanza != "" && f.Stanza != "-" {
				ret = append(ret, f)
			}
		} else {
			s := structs[f.Type.PackageID][f.Type.Name]
			ret = append(ret, s.FlatFields(structs)...)
		}
	}
	return ret
}

func (p *parser) parseType(pack *packages.Package, typeExpr ast.Expr, file *ast.File) *_type {
	switch t := typeExpr.(type) {
	case *ast.Ident:
		return &_type{
			Name: t.Name,
		}
	case *ast.ArrayType:
		e := p.parseType(pack, t.Elt, file)
		if e == nil {
			return nil
		}
		return &_type{
			ArrayEltType: e,
		}
	case *ast.MapType:
		k := p.parseType(pack, t.Key, file)
		if k == nil {
			return nil
		}
		v := p.parseType(pack, t.Value, file)
		if v == nil {
			return nil
		}
		return &_type{
			MapKeyType:   k,
			MapValueType: v,
		}
	case *ast.SelectorExpr:
		x := p.parseType(pack, t.Sel, file)
		if x == nil {
			return nil
		}
		for _, v := range pack.Imports {
			if v.Name == t.X.(*ast.Ident).Name {
				x.PackageID = v.ID
				return x
			}
		}
		for _, v := range file.Imports {
			if v.Name != nil && v.Name.Name == t.X.(*ast.Ident).Name {
				x.PackageID = pack.Imports[strings.ReplaceAll(v.Path.Value, "\"", "")].ID
				return x
			}
		}
		if x.PackageID == "" {
			fmt.Printf("Error looking up package in imports %s\n", t.X.(*ast.Ident).Name)
			for n := range pack.Imports {
				fmt.Printf("  %s\n", n)
			}
		}
		return nil
	case *ast.StarExpr:
		x := p.parseType(pack, t.X, file)
		if x == nil {
			return nil
		}
		x.IsPointer = true
		return x
	default:
		return nil
	}
}

func (p *parser) parseField(pack *packages.Package, name string, field *ast.Field, file *ast.File) *_field {
	t := p.parseType(pack, field.Type, file)
	if t == nil {
		return nil
	}
	f := &_field{
		Name:   name,
		Stanza: getTagValue(field.Tag, `json:\"([^,\"]*).*\"`),
		Desc:   strings.ReplaceAll(strings.TrimSpace(field.Doc.Text()), "\n", "<br />"),
		Type:   t,
	}
	if !(f.Type.ArrayEltType != nil || f.Type.MapKeyType != nil || f.Type.MapValueType != nil || f.Type.IsPointer) {
		f.Required = !strings.Contains(getTagValue(field.Tag, `json:\"(.*)\"`), "omitempty")
	}
	return f
}

func (p *parser) parseStruct(pack *packages.Package, typeSpec *ast.TypeSpec, doc *ast.CommentGroup, file *ast.File) _struct {
	s := _struct{
		Desc: strings.ReplaceAll(strings.TrimSpace(doc.Text()), "\n", "<br />"),
	}

	structure, ok := typeSpec.Type.(*ast.StructType)
	if ok {
		for _, field := range structure.Fields.List {
			if len(field.Names) == 0 {
				f := p.parseField(pack, "", field, file)
				if f != nil {
					s.Fields = append(s.Fields, f)
				}
			} else {
				for _, name := range field.Names {
					f := p.parseField(pack, name.Name, field, file)
					if f != nil {
						f.IsExported = name.IsExported()
						s.Fields = append(s.Fields, f)
					}

				}
			}
		}
	}

	return s
}

type parser struct {
	packages []*packages.Package
	structs  map[string]map[string]_struct
}

func (p *parser) parsePackage(pack *packages.Package) {
	if _, ok := p.structs[pack.ID]; !ok {
		p.structs[pack.ID] = make(map[string]_struct)
		for _, v := range pack.Imports {
			p.parsePackage(v)
		}
		for _, node := range pack.Syntax {
			for _, decl := range node.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if ok {
					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if ok {
							s := p.parseStruct(pack, typeSpec, genDecl.Doc, node)
							s.IsExported = typeSpec.Name.IsExported()
							p.structs[pack.ID][typeSpec.Name.Name] = s
						}
					}
				}
			}
		}
	}
}

func (p *parser) fixLocalTypes(pack string, t *_type) {
	if t.PackageID == "" {
		if _, ok := p.structs[pack][t.Name]; ok {
			t.PackageID = pack
		}
	}
	if t.ArrayEltType != nil {
		p.fixLocalTypes(pack, t.ArrayEltType)
	}
	if t.MapKeyType != nil {
		p.fixLocalTypes(pack, t.MapKeyType)
	}
	if t.MapValueType != nil {
		p.fixLocalTypes(pack, t.MapValueType)
	}
}

func (p *parser) parsePackages() {
	for _, pack := range p.packages {
		p.parsePackage(pack)
	}
	for pack, r := range p.structs {
		for _, s := range r {
			for _, f := range s.Fields {
				p.fixLocalTypes(pack, f.Type)
			}
		}
	}
}

const tmpl = `# Package {{ .Package }}

{{ range $name, $struct := .Structs -}}
- [{{ $name }}](#{{ $name }})
{{ end }}

{{ range $name, $struct := .Structs -}}
## {{ $name }}

{{ if $struct.Desc }}{{ $struct.Desc }}{{ end }}

{{ if $struct.Fields -}}

| Stanza | Type | Required | Description |
|---|---|---|---|
{{- range $struct.Fields }}
| {{ if .Stanza }}` + "`" + `{{ .Stanza }}` + "`" + `{{ end }} | {{ .Type.Format }} | {{ if .Required }}Yes{{ else }}No{{ end }} | {{ .Desc }} |
{{- end }}
{{- end }}

{{ end }}
`

func (p *parser) processType(t *_type, ret map[string]map[string]_struct) bool {
	if t.PackageID != "" {
		if _, ok := p.structs[t.PackageID][t.Name]; ok {
			if _, ok := ret[t.PackageID]; !ok {
				ret[t.PackageID] = make(map[string]_struct)
			}
			if _, ok := ret[t.PackageID][t.Name]; !ok {
				s := p.structs[t.PackageID][t.Name]
				s.Fields = s.FlatFields(p.structs)
				ret[t.PackageID][t.Name] = s
				return true
			}
		}
	}
	if t.ArrayEltType != nil {
		if p.processType(t.ArrayEltType, ret) {
			return true
		}
	}
	if t.MapKeyType != nil {
		if p.processType(t.MapKeyType, ret) {
			return true
		}
	}
	if t.MapValueType != nil {
		if p.processType(t.MapValueType, ret) {
			return true
		}
	}
	return false
}

func (p *parser) extract() ([]string, map[string]map[string]_struct) {
	packs := []string{}
	for _, r := range p.packages {
		packs = append(packs, r.ID)
	}
	structs := make(map[string]map[string]_struct)
	for _, pack := range packs {
		structs[pack] = make(map[string]_struct)
		for n, s := range p.structs[pack] {
			s.Fields = s.FlatFields(p.structs)
			if len(*roots) > 0 {
				for _, check := range *roots {
					if check == n {
						structs[pack][n] = s
					}
				}
			} else {
				structs[pack][n] = s
			}
		}
	}
	for {
		changed := false
		for _, pack := range structs {
			for _, s := range pack {
				for _, f := range s.Fields {
					if f.IsExported && f.Stanza != "" && f.Stanza != "-" {
						changed = changed || p.processType(f.Type, structs)
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	return packs, structs
}

func packageToFile(pack string) string {
	fn := strings.ReplaceAll(pack, "/", "-")
	fn = strings.ReplaceAll(fn, ".", "-")
	return fn + ".md"
}

func print(packs []string, structs map[string]map[string]_struct, output string) {
	t := template.New("struct-docs")
	if _, err := t.Parse(tmpl); err != nil {
		panic(err)
	}
	for k, v := range structs {
		if len(v) > 0 {
			f, err := os.Create(path.Join(output, packageToFile(k)))
			if err != nil {
				panic(err)
			}
			if err := t.Execute(f, map[string]interface{}{
				"Package": k,
				"Structs": v,
			}); err != nil {
				panic(err)
			}
		}
	}
}

func main() {
	pflag.Parse()

	if *help {
		pflag.PrintDefaults()
		return
	}

	if inputs == nil || len(*inputs) == 0 {
		fmt.Println("You must specify at least one input")
		pflag.PrintDefaults()
		return
	}
	if outputPath == nil || *outputPath == "" {
		fmt.Println("You must specify an output path.")
		pflag.PrintDefaults()
		return
	}

	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	packs, _ := packages.Load(&cfg, *inputs...)

	parser := parser{
		packages: packs,
		structs:  make(map[string]map[string]_struct),
	}

	parser.parsePackages()

	roots, structs := parser.extract()

	if err := os.MkdirAll(*outputPath, os.ModePerm); err != nil {
		panic(err)
	}

	print(roots, structs, *outputPath)
}
