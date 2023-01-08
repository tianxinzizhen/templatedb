package xml

import (
	"embed"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/tianxinzizhen/templatedb/template"
)

type Sql struct {
	Func      string `xml:"func,attr"`
	Name      string `xml:"name,attr"`
	Common    bool   `xml:"common,attr"`
	Statement string `xml:",chardata"`
}

type SqlStatementRoot struct {
	XMLName xml.Name `xml:"root"`
	Pkg     string   `xml:"pkg,attr"`
	Sql     []Sql    `xml:"sql"`
}

func LoadTemplateStatements(sqlDir embed.FS, template map[string]*template.Template, parse func(parse string, addParseTrees ...func(*template.Template) error) (*template.Template, error)) error {
	dir, err := sqlDir.ReadDir(".")
	if err != nil {
		return err
	}
	dirName := dir[0].Name()
	files, err := sqlDir.ReadDir(dir[0].Name())
	if err != nil {
		return err
	}
	for _, fileInfo := range files {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), "_sql.xml") {
			bytes, err := sqlDir.ReadFile(dirName + "/" + fileInfo.Name())
			if err != nil {
				return err
			}
			sqlRoot := SqlStatementRoot{}
			xml.Unmarshal(bytes, &sqlRoot)
			addParseTree := addCommonTemplate(sqlRoot.Sql, parse)
			for _, v := range sqlRoot.Sql {
				key := fmt.Sprintf("%s.%s:%s", sqlRoot.Pkg, v.Func, v.Name)
				template[key], err = parse(v.Statement, addParseTree)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addCommonTemplate(sqls []Sql, parse func(parse string, addParseTrees ...func(*template.Template) error) (*template.Template, error)) func(*template.Template) error {
	return func(t *template.Template) error {
		for _, v := range sqls {
			if v.Common {
				template, err := parse(v.Statement)
				if err != nil {
					return err
				}
				t.AddParseTree(v.Name, template.Tree)
			}
		}
		return nil
	}
}
