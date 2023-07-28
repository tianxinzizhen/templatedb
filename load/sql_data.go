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
	Common     bool
	ParamMap   map[string]int
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
	for _, v := range astComment.Decls {
		if genDecl, ok := v.(*ast.GenDecl); ok {
			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							for _, field := range structType.Fields.List {
								if fc, ok := field.Type.(*ast.FuncType); ok {
									sqlDataInfo := &SqlDataInfo{
										Name:     field.Names[0].String(),
										FuncName: fmt.Sprintf("%s.%s.%s:", pkg, typeSpec.Name.String(), field.Names[0].String()),
									}
									for _, ci := range field.Doc.List {
										if strings.HasPrefix(ci.Text, "//sql") {
											sqlDataInfo.Sql = ci.Text[5:]
										}
										if strings.HasPrefix(ci.Text, "/*sql") {
											sqlDataInfo.Sql = ci.Text[5 : len(ci.Text)-2]
										}
										if strings.HasPrefix(ci.Text, "//not-prepare") {
											sqlDataInfo.NotPrepare = true
										}
										if strings.HasPrefix(sqlDataInfo.Sql, ":not-prepare") {
											sqlDataInfo.NotPrepare = true
											sqlDataInfo.Sql = sqlDataInfo.Sql[len(":not-prepare"):]
										}
										if strings.HasPrefix(sqlDataInfo.Sql, ":common") {
											sqlDataInfo.Common = true
											sqlDataInfo.Sql = sqlDataInfo.Sql[len(":common"):]
										}
									}
									if fc.Params != nil && len(fc.Params.List) > 0 && len(fc.Params.List[0].Names) > 0 {
										sqlDataInfo.ParamMap = make(map[string]int)
										for i, v := range fc.Params.List {
											sqlDataInfo.ParamMap[v.Names[0].Name] = i
										}
									}
									sqlDataInfos = append(sqlDataInfos, sqlDataInfo)
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
