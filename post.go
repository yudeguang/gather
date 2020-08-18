// Copyright 2020 ratelimit Author(https://github.com/yudeguang/gather). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/gather.
//模拟浏览器进行数据采集包,可较方便的定义http头，同时全自动化处理cookies
package gather

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
)

/*
post方式获取数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postMap:指post过去的相关数据

例:
ga:= NewGather("chrome", false)
postMap := make(map[string]string)
postMap["user"] = "ydg"
postMap["password"] = "abcdef"
html, redirectURL, err := ga.Post("https://weibo.com/xxxxx", "", postMap)
*/
func (g *GatherStruct) Post(URL, refererURL string, postMap map[string]string) (html, redirectURL string, err error) {
	return g.PostUtil(URL, refererURL, "", postMap)
}

/*
post方式获取数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postMap:指post过去的相关数据
例:
ga := NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postMap := make(map[string]string)
postMap["user"] = "ydg"
postMap["password"] = "abcdef"
html, redirectURL, err := ga.PostUtil("https://weibo.com/xxxxx", "",cookies, postMap)
*/
func (g *GatherStruct) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	postValues := url.Values{}
	for k, v := range postMap {
		postValues.Set(k, v)
	}
	postDataStr := postValues.Encode()
	postDataBytes := []byte(postDataStr)
	postBytesReader := bytes.NewReader(postDataBytes)
	if _, eixst := g.safeHeaders.Load("Content-Type"); !eixst {
		g.safeHeaders.Store("Content-Type", "application/x-www-form-urlencoded; param=value")
	}
	req, err := g.newHttpRequest("POST", URL, refererURL, cookies, postBytesReader)
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

//POST二进制
func (g *GatherStruct) PostBytes(URL, refererURL, cookies string, postBytes []byte) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	postBytesReader := bytes.NewReader(postBytes)
	req, err := g.newHttpRequest("POST", URL, refererURL, cookies, postBytesReader)
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

/*
以XML的方式post数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postXML:指待Post的XML数据，文本类型
例:
ga := gather.NewGather("chrome", false)
postXML := `<?xml version="1.0" encoding="utf-8"?><loin><user>ydg</user><passord>abcdef</passord></loin>`
html, redirectURL, err := ga.PostXML(`https://weibo.com/xxxxx`, "", postXML)
*/
func (g *GatherStruct) PostXML(URL, refererURL, postXML string) (html, redirectURL string, err error) {
	return g.PostXMLUtil(URL, refererURL, "", postXML)
}

/*
以XML的方式post数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
postXML:指待Post的XML数据，文本类型

例:
ga := gather.NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postXML := `<?xml version="1.0" encoding="utf-8"?><loin><user>ydg</user><passord>abcdef</passord></loin>`
html, redirectURL, err := ga.PostXML(`https://weibo.com/xxxxx`, "", cookies, postXML)
*/
func (g *GatherStruct) PostXMLUtil(URL, refererURL, cookies, postXML string) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	g.safeHeaders.Store("Content-Type", "application/xml")
	req, err := g.newHttpRequest("POST", URL, refererURL, cookies, strings.NewReader(postXML))
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

/*
以json的方式post数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postJson:指待Post的json数据，文本类型

例:
ga := gather.NewGather("chrome", false)
postJson := `{"user":"ydg","password":"abcdesg"}`
html, redirectURL, err := ga.PostJson(`https://weibo.com/xxxxx`, "", postJson)
*/
func (g *GatherStruct) PostJson(URL, refererURL, postJson string) (html, redirectURL string, err error) {
	return g.PostJsonUtil(URL, refererURL, "", postJson)
}

/*
以json的方式post数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
postJson:指待Post的json数据，文本类型

例:
ga := gather.NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postJson := `{"user":"ydg","password":"abcdesg"}`
html, redirectURL, err := ga.PostJsonUtil(`https://weibo.com/xxxxx`, "", cookies, postJson)
*/
func (g *GatherStruct) PostJsonUtil(URL, refererURL, cookies, postJson string) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	g.safeHeaders.Store("Content-Type", "application/json")
	req, err := g.newHttpRequest("POST", URL, refererURL, cookies, strings.NewReader(postJson))
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

//multipart/form-data 上传文件的结构体
type multipartPostFile struct {
	fileName    string
	contentType string
	content     []byte
}

//multipart/form-data方式POST数据,手动增加cookies
func (g *GatherStruct) PostMultipartformData(URL, refererURL, cookies, boundary string, postValueMap map[string]string, postFileMap map[string]multipartPostFile) (html, redirectURL string, err error) {
	return g.PostMultipartformDataUtil(URL, refererURL, "", boundary, postValueMap, postFileMap)
}

//multipart/form-data方式POST数据,自动继承先前的cookies
//boundary指post“分割边界”,这个“边界数据”不能在内容其他地方出现,一般来说使用一段从概率上说“几乎不可能”的数据即可
//postValueMap指post的普通文本,只包含name和value
//postFileMap指上传的文件,比如图片,需在调用此函数前自行转换成[]byte,当然POST协议也可使用base64编码后,不过在此忽略此用法,base64也请转换成[]byte
//multipart/form-data数据格式参见标准库中： mime\multipart\testdata\nested-mime,注意此处file文件是用的base64编码后的
func (g *GatherStruct) PostMultipartformDataUtil(URL, refererURL, cookies, boundary string, postValueMap map[string]string, postFileMap map[string]multipartPostFile) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	if boundary == "" {
		boundary = `--WebKitFormBoundaryTP3TumA8yjBZCv2R`
	}
	postData := ``
	for name, value := range postValueMap {
		postData = postData + boundary + "\r\n" +
			`Content-Disposition: form-data; name="` + name + `"` + "\r\n\r\n" + value
	}
	for name, onePostFile := range postFileMap {
		postData = postData + boundary + "\r\n" +
			`Content-Disposition: form-data; name="` + name + `"; filename="` + onePostFile.fileName + `"` + "\r\n" +
			`Content-Type: ` + onePostFile.contentType + "\r\n\r\n" +
			string(onePostFile.content)
	}
	postData = postData + "\r\n" + boundary + `--`
	g.safeHeaders.Store("Content-Type", "multipart/form-data; boundary="+boundary)
	req, err := http.NewRequest("POST", URL, strings.NewReader(postData))
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}
