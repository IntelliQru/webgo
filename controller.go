package webgo

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	//"path/filepath"
	"strconv"

	"github.com/IntelliQru/i18n"
)

type (
	Controller struct {
		Ctx *Context
		T   i18n.TFuncHandler
	}
	ControllerInterface interface {
		Init(ctx *Context)
		Prepare() bool
		Finish()
		Error(code int, tpl string)

		GetHeader(key string) string
		SetHeader(key string, val string)
		SetStatusCode(code int)
		SetOutput(data []byte)

		Redirect(location string, code int)

		SendFile(filepath string) (err error)
		Render(tpl_name string, data interface{})
		Json(data interface{}, unicode bool)
		Plain(data string)

		Error504(tpl string)

		exec()
	}
)

func (c *Controller) Error504(tpl string) {
	if tpl == "" {
		tpl = "504 Gateway Timeout"
	}
	c.Ctx.output = []byte(tpl)
	c.Ctx.code = 504
}

func (c *Controller) Init(ctx *Context) {
	c.Ctx = ctx
	c.T = i18n.Tfunc(c.Ctx.Lang)
}
func (c Controller) Prepare() bool {
	return true
}
func (c Controller) Finish() {}
func (c Controller) Error(code int, data string) {
	c.Ctx.output = []byte(data)
	c.Ctx.code = code
}

func (c Controller) GetHeader(key string) string {
	return c.Ctx.Request.Header.Get(key)
}
func (c Controller) SetHeader(key string, val string) {
	c.Ctx.Response.Header().Set(key, val)
}
func (c Controller) SetStatusCode(code int) {
	c.Ctx.code = code
}
func (c Controller) SetOutput(data []byte) {
	c.Ctx.output = data
}
func (c Controller) Redirect(location string, code int) {
	c.SetStatusCode(code)
	c.SetHeader("Location", location)
}



func (c Controller) SendFile(path string) (err error) {

	c.CustomResponse()

	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return
	}

	FileHeader := make([]byte, 512)
	file.Read(FileHeader)
	FileContentType := http.DetectContentType(FileHeader)

	// Получаем размер файла
	FileStat, _ := file.Stat()
	FileSize := strconv.FormatInt(FileStat.Size(), 10)

	//Filename := filepath.Base(path)

	//c.Ctx.Response.Header().Set("Content-Disposition", "attachment; filename="+Filename)
	c.Ctx.Response.Header().Set("Content-Type", FileContentType)
	c.Ctx.Response.Header().Set("Content-Length", FileSize)

	file.Seek(0, 0)
	io.Copy(c.Ctx.Response, file)

	return
}

func (c Controller) CustomResponse() {
	c.Ctx.isCustomResponse = true
}

func (c Controller) Render(tpl_name string, data interface{}) {
	bytes := bytes.NewBufferString("")
	c.Ctx.error = app.templates.ExecuteTemplate(bytes, tpl_name+".html", data)
	if c.Ctx.error != nil {
		return
	}
	c.Ctx.output, c.Ctx.error = ioutil.ReadAll(bytes)
}

func (c Controller) Json(data interface{}, unicode bool) {
	var content []byte
	c.Ctx.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Ctx.output, c.Ctx.error = json.Marshal(data)
	if c.Ctx.error != nil {
		return
	}

	if !unicode {
		return
	}

	rs := []rune(string(content))
	jsons := ""
	for _, r := range rs {
		rint := int(r)
		if rint < 128 {
			jsons += string(r)
		} else {
			jsons += "\\u" + strconv.FormatInt(int64(rint), 16)
		}
	}
	c.Ctx.output = []byte(jsons)
}

func (c Controller) Plain(data string) {
	c.Ctx.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Ctx.output = []byte(data)
}

func (c Controller) exec() {
	if c.Ctx.error != nil {
		LOGGER.Error(c.Ctx.error)
		if c.Ctx.code == 0 {
			c.Ctx.code = 500
		}
		c.Ctx.Response.WriteHeader(c.Ctx.code)
		c.Ctx.Response.Write(c.Ctx.output)
		return
	}

	// Проверяем редирект
	if c.Ctx.IsRedirect() {
		c.Ctx.Response.WriteHeader(c.Ctx.code)
		return
	}

	// Выводим данные
	if c.Ctx.code == 0 {
		c.Ctx.code = 200
	}

	if c.Ctx.isCustomResponse {
		return
	}
	c.Ctx.Response.WriteHeader(c.Ctx.code)
	c.Ctx.Response.Write(c.Ctx.output)
	return
}
