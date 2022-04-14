package traefik_plugin_check

import (
	"bytes"
	"context"
	"mime"
	"net/http"
	"regexp"
	"unicode/utf8"
)

// Config the plugin configuration.
type Config struct {
	CheckMediaType    string
	CheckCharSet      string
	CheckHeader       string
	CheckHeaderRegExp string
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		CheckMediaType:    "",
		CheckCharSet:      "",
		CheckHeader:       "",
		CheckHeaderRegExp: "",
	}
}

// Demo a Demo plugin.
type Checker struct {
	next http.Handler
	name string
	conf *Config
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Checker{
		next: next,
		name: name,
		conf: config,
	}, nil
}

func (c *Checker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	rww := &ResponseWriterWraper{
		w:    rw,
		cmt:  c.conf.CheckMediaType,
		cs:   c.conf.CheckCharSet,
		buf:  &bytes.Buffer{},
		hd:   rw.Header().Clone(),
		code: http.StatusForbidden,
		xh:   c.conf.CheckHeader,
		xhr:  c.conf.CheckHeaderRegExp,
	}

	c.next.ServeHTTP(rww, req)
}

type ResponseWriterWraper struct {
	w    http.ResponseWriter
	cmt  string
	cs   string
	code int
	hd   http.Header
	buf  *bytes.Buffer
	xh   string
	xhr  string
}

func (rww *ResponseWriterWraper) Write(p []byte) (int, error) {
	num, err := rww.buf.Write(p)
	mediatype, params, err := mime.ParseMediaType(rww.hd.Get("Content-Type"))
	matched, err := regexp.Match(rww.xhr, []byte(rww.hd.Get(rww.xh)))
	if matched {
		if rww.cmt == mediatype {
			if params["charset"] == rww.cs {
				switch rww.cs {
				case "utf-8":
					if utf8.ValidString(rww.buf.String()) {
						headerClone(rww.hd, rww.w.Header())
						rww.w.Write(p)
						rww.w.WriteHeader(http.StatusOK)
					}
				case "gbk":
					if isGBK(rww.buf.Bytes()) {
						headerClone(rww.hd, rww.w.Header())
						rww.w.Write(p)
						rww.w.WriteHeader(http.StatusOK)
					}
				case "gb2312":
					if isGBK(rww.buf.Bytes()) {
						headerClone(rww.hd, rww.w.Header())
						rww.w.Write(p)
						rww.w.WriteHeader(http.StatusOK)
					}
				default:
					rww.w.WriteHeader(http.StatusBadRequest)
				}
			}
		} else {
			rww.w.WriteHeader(http.StatusNotFound)
		}
	}
	rww.w.WriteHeader(http.StatusConflict)
	return num, err
}

//把Write前所有的数据都存下来
func (rww *ResponseWriterWraper) Header() http.Header {
	return rww.hd
}

//都存下来
func (rww *ResponseWriterWraper) WriteHeader(i int) {
	rww.code = i
	return
}

func isGBK(data []byte) bool {
	length := len(data)
	var i int = 0
	for i < length {
		//fmt.Printf("for %x\n", data[i])
		if data[i] <= 0xff {
			//编码小于等于127,只有一个字节的编码，兼容ASCII吗
			i++
			continue
		} else {
			//大于127的使用双字节编码
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

func headerClone(source http.Header, target http.Header) {
	for k, vv := range source {
		for _, v := range vv {
			target.Add(k, v)
		}
	}
	return
}