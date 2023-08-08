package xml

import (
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/tianxinzizhen/templatedb/template"
)

type Sql struct {
	Func       string `xml:"func,attr"`
	Name       string `xml:"name,attr"`
	NotPrepare bool   `xml:"notPrepare,attr"`
	Param      string `xml:"param,attr"`
	Statement  string `xml:",chardata"`
}

type SqlStatementRoot struct {
	XMLName xml.Name `xml:"root"`
	Pkg     string   `xml:"pkg,attr"`
	Sql     []Sql    `xml:"sql"`
}

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
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".xml") {
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
		return errors.New("sql xml bytes is nil")
	}
	sqlRoot := SqlStatementRoot{}
	err := xml.Unmarshal(bytes, &sqlRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(pkg) == "" {
		pkg = sqlRoot.Pkg
	}
	for _, v := range sqlRoot.Sql {
		key := fmt.Sprintf("%s.%s:%s", pkg, v.Func, v.Name)
		template[key], err = parse(v.Statement)
		if err != nil {
			return err
		}
		template[key].NotPrepare = v.NotPrepare
		if len(v.Param) > 0 {
			for _, v := range strings.Split(v.Param, ",") {
				pname, _, _ := strings.Cut(v, " ")
				template[key].Param = append(template[key].Param, strings.TrimSpace(pname))
			}
		}
	}
	return nil
}

func LoadTemplateStatementsOfString(pkg, xmlSqls string, template map[string]*template.Template, parse func(parse string) (*template.Template, error)) error {
	return LoadTemplateStatementsOfBytes(pkg, []byte(xmlSqls), template, parse)
}
