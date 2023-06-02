package templatedb

import (
	"embed"
	"errors"

	commentStruct "github.com/tianxinzizhen/templatedb/load/comment"
	"github.com/tianxinzizhen/templatedb/load/xml"
	"github.com/tianxinzizhen/templatedb/template"
)

func (db *OptionDB) LoadXml(pkg string, sql any) (err error) {
	switch v := sql.(type) {
	case embed.FS:
		err = db.loadXmlFS(pkg, v)
	case string:
		err = db.loadXmlString(pkg, v)
	case []byte:
		err = db.loadXmlBytes(pkg, v)
	default:
		err = errors.New("xml sql type load data not support")
	}
	return
}
func (db *OptionDB) loadXmlFS(pkg string, sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return xml.LoadTemplateStatements(pkg, sqlfs, db.template, db.parse)
}

func (db *OptionDB) loadXmlString(pkg string, sql string) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return xml.LoadTemplateStatementsOfString(pkg, sql, db.template, db.parse)
}

func (db *OptionDB) loadXmlBytes(pkg string, sqlBytes []byte) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return xml.LoadTemplateStatementsOfBytes(pkg, sqlBytes, db.template, db.parse)
}

func (db *OptionDB) LoadComment(pkg string, sql any) (err error) {
	switch v := sql.(type) {
	case embed.FS:
		err = db.loadCommentFS(pkg, v)
	case string:
		err = db.loadCommentString(pkg, v)
	case []byte:
		err = db.loadCommentBytes(pkg, v)
	default:
		err = errors.New("comment sql type load data not support")
	}
	return
}

func (db *OptionDB) loadCommentFS(pkg string, sqlfs embed.FS) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return commentStruct.LoadTemplateStatements(pkg, sqlfs, db.template, db.parse)
}
func (db *OptionDB) loadCommentString(pkg string, sql string) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return commentStruct.LoadTemplateStatementsOfString(pkg, sql, db.template, db.parse)
}
func (db *OptionDB) loadCommentBytes(pkg string, sqlBytes []byte) error {
	if db.template == nil {
		db.template = make(map[string]*template.Template)
	}
	return commentStruct.LoadTemplateStatementsOfBytes("", sqlBytes, db.template, db.parse)
}
