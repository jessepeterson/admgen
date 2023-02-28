package admgen

import (
	"fmt"
	"go/ast"
	"go/token"
)

// Key represents the "key" type of the Apple Device Management YAML.
type Key struct {
	Key      string `yaml:"key"`
	Type     string `yaml:"type"`
	Presence string `yaml:"presence,omitempty"`
	SubKeys  []Key  `yaml:"subkeys,omitempty"`
	Content  string `yaml:"content"`
}

type DeclBuilder struct {
	Decls     *[]ast.Decl
	NeedsTime bool
}

func (b *DeclBuilder) Walk(keys []Key, name string) {
	b.handleDict(keys, name)
}

func (b *DeclBuilder) handleKey(key Key) (string, string) {
	switch key.Type {
	case "<string>":
		return "string", ""
	case "<boolean>":
		return "bool", ""
	case "<integer>":
		return "int", ""
	case "<data>":
		return "[]byte", ""
	case "<real>":
		return "float64", ""
	// case "<date>":
	// 	b.NeedsTime = true
	// 	return "time.Time", ""
	case "<dictionary>":
		name := normalizeFieldName(key.Key)
		if len(key.SubKeys) == 1 {
			switch key.SubKeys[0].Type {
			case "<dictionary>":
				b.handleDict(key.SubKeys[0].SubKeys, name)
				return "map[string]" + name, "assuming string map for single dictionary subkey"
			case "<any>":
				return "interface{}", "<any> type as single dictionary subkey"
			}
		}
		b.handleDict(key.SubKeys, name)
		return name, ""
	case "<array>":
		return b.handleArray(key.SubKeys)
	default:
		return "interface{}", fmt.Sprintf("unknown type: %s", key.Type)
	}
}

func (b *DeclBuilder) handleArray(keys []Key) (string, string) {
	if len(keys) < 1 {
		return "interface{}", "missing array keys in schema"
	}
	keyType := keys[0].Type
	for _, key := range keys[1:] {
		if key.Type != keyType {
			// return an interface{} as we seem to have mismatched types within our array
			return "interface{}", "mismatched array types in schema"
		}
	}
	goType, comment := b.handleKey(keys[0])
	if len(keys) == 1 && keys[0].Type != "<dictionary>" && len(keys[0].SubKeys) > 0 {
		// if our single key is a scalar type and we have subkeys
		// then the the subkeys describe actual array values
		if comment != "" {
			comment += ", "
		}
		comment += fmt.Sprintf("%d array values defined in schema", len(keys[0].SubKeys))
	}
	return "[]" + goType, comment
}

func (b *DeclBuilder) handleDict(keys []Key, name string) {
	fl := new(ast.FieldList)
	for _, key := range keys {
		goType, comment := b.handleKey(key)
		omitempty := ""
		if key.Presence == "optional" {
			omitempty = ",omitempty"
		}
		f := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(normalizeFieldName(key.Key))},
			Type:  ast.NewIdent(goType),
			Tag:   &ast.BasicLit{Value: "`plist:\"" + key.Key + omitempty + "\"`"},
		}
		if comment != "" {
			f.Comment = &ast.CommentGroup{List: []*ast.Comment{{Text: " // " + comment}}}
		}
		fl.List = append(fl.List, f)
	}
	decl := &ast.GenDecl{
		Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// " + name}}},
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: ast.NewIdent(name),
			Type: &ast.StructType{Fields: fl},
		}},
	}
	*b.Decls = append(*b.Decls, decl)
}
