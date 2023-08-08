package comment

import (
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/tianxinzizhen/templatedb/template"
)

func LoadTemplateStatements(pkg string, sqlDir embed.FS, template map[string]*template.Template, parse func(parse string) (*template.Template, error)) error {
	files, err := sqlDir.ReadDir(".")
	if err != nil {
		return err
	}
	dirName := ""
	if files[0].IsDir() {
		dirName = files[0].Name() + "/"
		files, err = sqlDir.ReadDir(files[0].Name())
		if err != nil {
			return err
		}
	}
	for _, fileInfo := range files {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".go") {
			bytes, err := sqlDir.ReadFile(dirName + fileInfo.Name())
			if err != nil {
				return err
			}
			err = LoadTemplateStatementsOfBytes(pkg, bytes, template, parse)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadTemplateStatementsOfBytes(pkg string, bytes []byte, template map[string]*template.Template, parse func(parse string) (*template.Template, error)) error {
	if bytes == nil {
		return errors.New("sql go bytes is nil")
	}
	astComment, err := parser.ParseFile(token.NewFileSet(), "", bytes, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, v := range astComment.Decls {
		if genDecl, ok := v.(*ast.GenDecl); ok {
			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							notDBFunc := true
							for _, field := range structType.Fields.List {
								if fv, ok := field.Type.(*ast.IndexExpr); ok {
									if sv, ok := (fv.X.(*ast.SelectorExpr)); ok && fmt.Sprint(sv) == "&{templatedb DBFunc}" {
										notDBFunc = false
									}
								}
							}
							if notDBFunc {
								continue
							}
							for _, field := range structType.Fields.List {
								if field.Doc != nil && len(field.Names) > 0 {
									key := fmt.Sprintf("%s.%s.%s:", pkg, typeSpec.Name.String(), field.Names[0].String())
									var sql string
									var notPrepare bool
									for _, ci := range field.Doc.List {
										if strings.HasPrefix(ci.Text, "//sql") {
											sql = ci.Text[5:]
										}
										if strings.HasPrefix(ci.Text, "/*sql") {
											sql = ci.Text[5 : len(ci.Text)-2]
										}
										if strings.HasPrefix(ci.Text, "//not-prepare") {
											notPrepare = true
										}
										if strings.HasPrefix(sql, ":not-prepare") {
											notPrepare = true
											sql = sql[len(":not-prepare"):]
										}
									}
									if len(sql) > 0 {
										template[key], err = parse(sql)
										if err != nil {
											return err
										}
										template[key].NotPrepare = notPrepare
										if fc, ok := field.Type.(*ast.FuncType); ok {
											if fc.Params != nil && len(fc.Params.List) > 0 && len(fc.Params.List[0].Names) > 0 {
												for _, v := range fc.Params.List {
													for _, v := range v.Names {
														template[key].Param = append(template[key].Param, v.Name)
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func LoadTemplateStatementsOfString(pkg string, sqlComments string, template map[string]*template.Template, parse func(parse string) (*template.Template, error)) error {
	return LoadTemplateStatementsOfBytes(pkg, []byte(sqlComments), template, parse)
}
