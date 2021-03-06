package webgo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Files []File
type File struct {
	Path string
	Name string
	Size int64
}

func (f *Files) RemoveAll() (err error) {
	for _, file := range *f {
		e := os.Remove(file.Path)
		if e != nil && err == nil {
			err = e
		}
	}

	return
}

type Context struct {
	Response    http.ResponseWriter
	Request     *http.Request
	Action      string
	Query       map[string]interface{}
	Files       Files
	Params      map[string]string
	_Body       []byte
	Body        map[string]interface{}
	User        map[string]interface{}
	code        int
	output      []byte
	Method      string
	ContentType string
	error       error
	close       bool
	isCustomResponse  bool // Это костыль, нужно переделать, чтобы не поломать АПИ текущих проектов
	Lang        string
}

func (c *Context) GetBody() []byte {
	return c._Body
}

func (c *Context) GetCookie(key string) string {
	val, err := c.Request.Cookie(key)
	if err != nil {
		return ""
	}
	return val.Value
}

// Порядок params - MaxAge, Path, Domain, HttpOnly, Secure
// Внимание! HttpOnly для сессий необходимо передавать true!!! Это органичет доступ к кукам JS в браузере
func (c *Context) SetCookie(name string, val string, params ...interface{}) {
	var cookie bytes.Buffer

	// Очищаем спец символы
	nameCleaner := strings.NewReplacer("\n", "-", "\r", "-")
	name = nameCleaner.Replace(name)

	valueCleaner := strings.NewReplacer("\n", " ", "\r", " ", ";", " ")
	val = valueCleaner.Replace(val)

	fmt.Fprintf(&cookie, "%s=%s", name, val)

	ln := len(params)

	if ln > 0 {
		var maxAge int64

		switch v := params[0].(type) {
		case int:
			maxAge = int64(v)
		case int32:
			maxAge = int64(v)
		case int64:
			maxAge = v
		}

		if maxAge > 0 {
			fmt.Fprintf(&cookie, "; Expires=%s; Max-Age=%d", time.Now().Add(time.Duration(maxAge)*time.Second).UTC().Format(time.RFC1123), maxAge)
		} else {
			fmt.Fprintf(&cookie, "; Max-Age=0")
		}
	}

	// Устанавливаем Path
	if ln > 1 {
		if v, ok := params[1].(string); ok && v != "" {
			fmt.Fprintf(&cookie, "; Path=%s", valueCleaner.Replace(v))
		}
	} else {
		fmt.Fprintf(&cookie, "; Path=%s", "/")
	}

	// Устанавливаем Domain
	if ln > 2 {
		if v, ok := params[2].(string); ok && v != "" {
			fmt.Fprintf(&cookie, "; Domain=%s", valueCleaner.Replace(v))
		}
	}

	// Устанавливаем HttpOnly
	if ln > 3 {
		if v, ok := params[3].(bool); ok && v {
			fmt.Fprintf(&cookie, "; HttpOnly")
		}
	}

	// Устанавливаем Secure
	if ln > 4 {
		var secure bool
		switch v := params[4].(type) {
		case bool:
			secure = v
		default:
			if params[4] != nil {
				secure = true
			}
		}

		if secure {
			fmt.Fprintf(&cookie, "; Secure")
		}

	}

	c.Response.Header().Add("Set-Cookie", cookie.String())
}

func (c *Context) ValidateSchema(schema interface{}) (err error) {

	switch c.ContentType {
	case CT_JSON:
		err = json.Unmarshal(c._Body, schema)
	case CT_FORM, CT_MULTIPART:

		schemaType := reflect.TypeOf(schema)

		if schemaType.Kind() == reflect.Ptr {
			schemaType = reflect.ValueOf(schema).Elem().Type()
		}

		if schemaType.Kind() != reflect.Struct {
			err = errors.New("Invalid validation struct type: " + schemaType.Kind().String())
			return
		}

		schemaValue := reflect.ValueOf(schema).Elem()

		for key, iVal := range c.Body {

			val := iVal.([]string)

			if len(val) == 0 {
				continue
			}

			field := schemaValue.FieldByName(key)

			if field.IsValid() {

				if field.Kind() == reflect.Slice {

					// Get kind of slice elements type
					arrElemKind := field.Type().Elem().Kind()

					for _, inValue := range val {

						switch arrElemKind {
						case reflect.String:
							field.Set(reflect.Append(field, reflect.ValueOf(inValue)))
						case reflect.Int:
							setVal, e := strconv.Atoi(inValue)
							if e != nil {
								err = errors.New("Invalid value '" + inValue + "' for key '" + key + "', must be Integer")
								return
							}
							field.Set(reflect.Append(field, reflect.ValueOf(setVal)))
						case reflect.Float64:
							setVal, e := strconv.ParseFloat(inValue, 64)
							if e != nil {
								err = errors.New("Invalid value '" + inValue + "' for key '" + key + "', must be Float64")
								return
							}
							field.Set(reflect.Append(field, reflect.ValueOf(setVal)))
						default:
							err = errors.New("Unsupported field type: " + arrElemKind.String())
							return
						}
					}

				} else {
					if len(val) > 1 {
						err = errors.New("Invalid array value for key '" + key + "'")
						return
					}

					fieldKind := field.Kind()

					switch fieldKind {
					case reflect.String:
						field.SetString(val[0])
					case reflect.Int:
						setVal, e := strconv.Atoi(val[0])
						if e != nil {
							err = errors.New("Invalid value '" + val[0] + "' for key '" + key + "', must be Integer")
							return
						}
						field.SetInt(int64(setVal))
					case reflect.Float64:
						setVal, e := strconv.ParseFloat(val[0], 64)
						if e != nil {
							err = errors.New("Invalid value '" + val[0] + "' for key '" + key + "', must be Float64")
							return
						}
						field.SetFloat(setVal)
					default:
						err = errors.New("Unsupported field type: " + fieldKind.String())
						return
					}
				}
			}
		}
	default:
		return errors.New("Invalid content type")
	}

	return
}

func (c *Context) IsRedirect() bool {
	return c.code == 301 || c.code == 302 || c.code == 303 || c.code == 307
}

func (c *Context) isString(val interface{}) bool {
	return false
}
func (c *Context) isInteger(val interface{}) bool {
	return false
}
func (c *Context) isMap(val interface{}) bool {
	return false
}
func (c *Context) isSlice(val interface{}) bool {
	return false
}
func (c *Context) isBool(val interface{}) bool {
	return false
}
