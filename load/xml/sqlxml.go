package xml

import (
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/tianxinzizhen/templatedb/load"
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

func LoadTemplateStatements(sqlDir embed.FS, template map[string]*template.Template, parse func(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error)) error {
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
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".xml") {
			bytes, err := sqlDir.ReadFile(dirName + fileInfo.Name())
			if err != nil {
				return err
			}
			sqlRoot := SqlStatementRoot{}
			err = xml.Unmarshal(bytes, &sqlRoot)
			if err != nil {
				return err
			}
			addParseTree := addCommonTemplate(sqlRoot.Sql, parse)
			for _, v := range sqlRoot.Sql {
				if !v.Common {
					key := fmt.Sprintf("%s.%s:%s", sqlRoot.Pkg, v.Func, v.Name)
					template[key], err = parse(v.Statement, addParseTree)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func LoadTemplateStatementsOfBytes(xmlSqls []byte, template map[string]*template.Template, parse func(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error)) error {
	if xmlSqls == nil {
		return errors.New("sql xml bytes is nil")
	}
	sqlRoot := SqlStatementRoot{}
	err := xml.Unmarshal([]byte(xmlSqls), &sqlRoot)
	if err != nil {
		return err
	}
	addParseTree := addCommonTemplate(sqlRoot.Sql, parse)
	for _, v := range sqlRoot.Sql {
		if !v.Common {
			key := fmt.Sprintf("%s.%s:%s", sqlRoot.Pkg, v.Func, v.Name)
			template[key], err = parse(v.Statement, addParseTree)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadTemplateStatementsOfString(xmlSqls string, template map[string]*template.Template, parse func(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error)) error {
	return LoadTemplateStatementsOfBytes([]byte(xmlSqls), template, parse)
}

func addCommonTemplate(sqls []Sql, parse func(parse string, addParseTrees ...load.AddParseTree) (*template.Template, error)) func(*template.Template) error {
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
