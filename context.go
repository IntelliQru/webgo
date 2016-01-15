package webgo

import (
	"net/http"
	"encoding/json"
	"fmt"
)

type Context struct {
	Response http.ResponseWriter
	Request *http.Request
	Output interface{}
	Query map[string]interface{}
	_Body []byte
	Body map[string]interface{}
	statusCode int
	body []byte
	ContentType string
}

func (c *Context) ValidateSchema (schema interface{}) (err error) {

	if c.ContentType == "application/x-www-form-urlencoded" {
		g:=c.Request.ParseForm()
		fmt.Println("",c.Request.PostForm,c.Request.Form, c.Request.PostFormValue("test"),g)
		// TODO: Реализовать
		return
	}

	if c.ContentType == "application/json" {
		err = json.Unmarshal(c._Body, schema)
		if err != nil {
			return
		}
		return
	}

	return
}