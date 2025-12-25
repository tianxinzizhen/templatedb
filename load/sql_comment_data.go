package load

import (
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type SqlDataInfo struct {
	FuncName   string
	Name       string
	Sql        string
	NotPrepare bool
	Batch      bool
	Param      []string
}

func LoadComment(pkg string, sql any) ([]*SqlDataInfo, error) {
	switch v := sql.(type) {
	case embed.FS:
		return LoadCommentEmbedFS(pkg, v)
	case string:
		return LoadCommentString(pkg, v)
	case []byte:
		return LoadCommentBytes(pkg, v)
	default:
		return nil, errors.New("comment sql type load data not support")
	}
}
func LoadCommentEmbedFS(pkg string, sqlDir embed.FS) ([]*SqlDataInfo, error) {
	files, err := sqlDir.ReadDir(".")
	if err != nil {
		return nil, err
	}
	dirName := ""
	if files[0].IsDir() {
		dirName = files[0].Name() + "/"
		files, err = sqlDir.ReadDir(files[0].Name())
		if err != nil {
			return nil, err
		}
	}
	var sqlDataInfos []*SqlDataInfo
	for _, fileInfo := range files {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".go") {
			bytes, err := sqlDir.ReadFile(dirName + fileInfo.Name())
			if err != nil {
				return nil, err
			}
			infos, err := LoadCommentBytes(pkg, bytes)
			if err != nil {
				return nil, err
			}
			sqlDataInfos = append(sqlDataInfos, infos...)
		}
	}
	return sqlDataInfos, nil
}

func LoadCommentBytes(pkg string, bytes []byte) ([]*SqlDataInfo, error) {
	if bytes == nil {
		return nil, errors.New("sql go bytes is nil")
	}
	astComment, err := parser.ParseFile(token.NewFileSet(), "", bytes, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var sqlDataInfos []*SqlDataInfo
	nameUnique := map[string]struct{}{}
	for _, v := range astComment.Decls {
		if genDecl, ok := v.(*ast.GenDecl); ok {
			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							for _, field := range structType.Fields.List {
								if fc, ok := field.Type.(*ast.FuncType); ok && field.Doc != nil {
									for _, ci := range field.Doc.List {
										var sql string
										if strings.HasPrefix(ci.Text, "//sql") {
											sql = ci.Text[5:]
										} else if strings.HasPrefix(ci.Text, "/*sql") {
											sql = ci.Text[5 : len(ci.Text)-2]
										}
										if len(sql) == 0 {
											continue
										}
										sqlDataInfo := &SqlDataInfo{
											Name:     field.Names[0].String(),
											FuncName: fmt.Sprintf("%s.%s.%s:", pkg, typeSpec.Name.String(), field.Names[0].String()),
											Sql:      sql,
										}
										var optionStr string
										if strings.HasPrefix(sqlDataInfo.Sql, "?option{") {
											if optionStr, sqlDataInfo.Sql, ok = strings.Cut(sqlDataInfo.Sql, "}"); ok {
												optionStr = strings.TrimSpace(optionStr)
												optionStr = strings.TrimPrefix(optionStr, "?option{")
												for _, v := range strings.Split(optionStr, ",") {
													v = strings.TrimSpace(v)
													if len(v) == 0 {
														continue
													}
													if k, v, ok := strings.Cut(v, ":"); ok {
														switch k {
														case "not_prepare":
															sqlDataInfo.NotPrepare = v == "true"
														case "batch":
															sqlDataInfo.Batch = v == "true"
														case "name":
															sqlDataInfo.Name = v
														}
													}
												}
											}
										}
										for _, v := range fc.Params.List {
											for _, v := range v.Names {
												if len(v.Name) > 0 {
													sqlDataInfo.Param = append(sqlDataInfo.Param, v.Name)
												}
											}
										}
										// 检查name是否重复
										if _, ok := nameUnique[sqlDataInfo.Name]; ok {
											return nil, fmt.Errorf("%s.%s load sql info by Duplicate name[%s]", pkg, typeSpec.Name.String(), sqlDataInfo.Name)
										} else {
											sqlDataInfos = append(sqlDataInfos, sqlDataInfo)
											nameUnique[sqlDataInfo.Name] = struct{}{}
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
	return sqlDataInfos, nil
}

func LoadCommentString(pkg string, sqlComments string) ([]*SqlDataInfo, error) {
	return LoadCommentBytes(pkg, []byte(sqlComments))
}
