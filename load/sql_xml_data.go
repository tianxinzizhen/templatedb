package load

import (
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

type Sql struct {
	Func       string `xml:"func,attr"`
	Name       string `xml:"name,attr"`
	NotPrepare bool   `xml:"notPrepare,attr"`
	Statement  string `xml:",chardata"`
}

type SqlStatementRoot struct {
	XMLName xml.Name `xml:"root"`
	Pkg     string   `xml:"pkg,attr"`
	Sql     []Sql    `xml:"sql"`
}

func LoadXml(pkg string, sql any) ([]*SqlDataInfo, error) {
	switch v := sql.(type) {
	case embed.FS:
		return LoadXMLEmbedFS(pkg, v)
	case string:
		return LoadXMLStrings(pkg, v)
	case []byte:
		return LoadXMLBytes(pkg, v)
	default:
		return nil, errors.New("comment sql type load data not support")
	}
}
func LoadXMLEmbedFS(pkg string, sqlDir embed.FS) ([]*SqlDataInfo, error) {
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
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".xml") {
			bytes, err := sqlDir.ReadFile(dirName + fileInfo.Name())
			if err != nil {
				return nil, err
			}
			infos, err := LoadXMLBytes(pkg, bytes)
			if err != nil {
				return nil, err
			}
			sqlDataInfos = append(sqlDataInfos, infos...)
		}
	}
	return sqlDataInfos, nil
}

func LoadXMLBytes(pkg string, bytes []byte) ([]*SqlDataInfo, error) {
	if bytes == nil {
		return nil, errors.New("sql xml bytes is nil")
	}
	sqlRoot := SqlStatementRoot{}
	err := xml.Unmarshal(bytes, &sqlRoot)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(pkg) == "" {
		pkg = sqlRoot.Pkg
	}
	var sqlDataInfos []*SqlDataInfo
	for _, v := range sqlRoot.Sql {
		sqlDataInfo := &SqlDataInfo{
			Name:       v.Func,
			FuncName:   fmt.Sprintf("%s.%s:%s", pkg, v.Func, v.Name),
			Sql:        v.Statement,
			NotPrepare: v.NotPrepare,
		}
		sqlDataInfos = append(sqlDataInfos, sqlDataInfo)
	}
	return sqlDataInfos, nil
}

func LoadXMLStrings(pkg, xmlSqls string) ([]*SqlDataInfo, error) {
	return LoadXMLBytes(pkg, []byte(xmlSqls))
}
