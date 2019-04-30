//模拟浏览器进行数据采集包,可较方便的定义http头，同时全自动化处理cookies
package gather

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

//内部变量全部大写导出，允许在执行过程中任意修改
type GatherStruct struct {
	Client  *http.Client
	Headers map[string]string
	J       *webCookieJar
}

/*
简单封装好的最常用的实例化采集器方法
Agent:指模拟的HTTP头
isCookieLogOpen:Cookie变更时是否打印

例:
ga := NewGather("baidu", false)
ga := NewGather("chrome", true)
*/
func NewGather(defaultAgent string, isCookieLogOpen bool) *GatherStruct {
	var headers = make(map[string]string)
	headers["User-Agent"] = defaultAgent
	return NewGatherUtil(headers, "", 300, isCookieLogOpen)
}

/*
简单封装好的含代理服务器的实例化采集器方法
Agent:指模拟的HTTP头
isCookieLogOpen:Cookie变更时是否打印
proxyURL:指代理服务器地址

例:
ga := NewGatherProxy("baidu", `https://104.207.139.207:8080`, false)
ga := NewGatherProxy("baidu", `https://104.207.139.207:8080`, true)
*/
func NewGatherProxy(defaultAgent string, proxyURL string, isCookieLogOpen bool) *GatherStruct {
	var headers = make(map[string]string)
	headers["User-Agent"] = defaultAgent
	return NewGatherUtil(headers, proxyURL, 300, isCookieLogOpen)
}

/*
最基础的实例化采集器
headers:指Request Headers
proxyURL:指代理服务器,不用则留空
timeOut:指抓取超时时间，以秒为单位
isCookieLogOpen:Cookie变更时是否打印
*/

// 例:
//  Headers := make(map[string]string)
//  Headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
// 	Headers["Accept-Encoding"] = "gzip, deflate, sdch"
// 	Headers["Accept-Language"] = "zh-CN,zh;q=0.8"
// 	Headers["Connection"] = "keep-alive"
// 	Headers["Upgrade-Insecure-Requests"] = "1"
// 	ga := gather.NewGatherUtil(Headers, "", 60, false)
func NewGatherUtil(headers map[string]string, proxyURL string, timeOut int, isCookieLogOpen bool) *GatherStruct {
	var gather GatherStruct
	gather.Headers = make(map[string]string)
	//先判断是不是从NewGather实例化而来,注意,此处排除用NewGatherUtil时只添加了一个User-Agent的情况,因为一般这种情况不存在
	if len(headers) == 1 {
		if v, exist := headers["User-Agent"]; exist {
			var defaultHeaders = make(map[string]string)
			defaultHeaders["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
			defaultHeaders["Accept-Encoding"] = "gzip, deflate, sdch"
			defaultHeaders["Accept-Language"] = "zh-CN,zh;q=0.8"
			defaultHeaders["Connection"] = "keep-alive"
			defaultHeaders["Upgrade-Insecure-Requests"] = "1"
			//User-Agent
			switch strings.ToLower(v) {
			case "baidu":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (compatible; Baiduspider/2.0;++http://www.baidu.com/search/spider.html)"
			case "google":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (compatible; Googlebot/2.1;+http://www.google.com/bot.html)"
			case "bing":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (compatible; bingbot/2.0;+http://www.bing.com/bingbot.htm)"
			case "chrome":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
			case "360":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/45.0.2454.101 Safari/537.36"
			case "ie", "ie9":
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Win64; x64; Trident/5.0)"
			case "": //默认
				defaultHeaders["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
			default:
				defaultHeaders["User-Agent"] = v
			}
			gather.Headers = defaultHeaders
		} else {
			gather.Headers = headers
		}
	} else {
		gather.Headers = headers
	}
	gather.J = newWebCookieJar(isCookieLogOpen)
	tr := &http.Transport{
		//TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true, //自动释放HTTP链接，以免启动多个和占用了所有端口
	}
	if proxyURL == "" {
		tr = &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: true,
		}
	} else {
		//设置代理服务器 proxyUrl 指类似 https://104.207.139.207:8080
		proxy := func(_ *http.Request) (*url.URL, error) { return url.Parse(proxyURL) }
		tr = &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: true,
			Proxy:              proxy,
		}
	}

	gather.Client = &http.Client{Transport: tr, Jar: gather.J}
	gather.Client.Timeout = time.Duration(timeOut) * time.Second
	return &gather
}

/*
GET方式获取数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL

例:
ga := NewGather("chrome", false)
html, returnedURL, err := ga.Get("https://www.baidu.com/", "")
*/
func (g *GatherStruct) Get(URL, refererURL string) (html, returnedURL string, err error) {
	return g.GetUtil(URL, refererURL, "")
}

/*
GET方式获取数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL

例:
ga:= NewGather("chrome", false)
cookies:=`SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
html, returnedURL, err := ga.GetUtil("https://weibo.com/xxxxxx",cookies, "")
*/
//GET方式获取数据,手动设置Cookie,Cookie留空则自动继承上次抓取时使用的Cookie
func (g *GatherStruct) GetUtil(URL, refererURL, cookies string) (html, returnedURL string, err error) {
	req, err := g.newHttpRequest("GET", URL, refererURL, cookies, nil)
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

/*
post方式获取数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postMap:指post过去的相关数据

例:
ga:= NewGather("chrome", false)
postMap := make(map[string]string)
postMap["user"] = "ydg"
postMap["password"] = "abcdef"
html, returnedURL, err := ga.Post("https://weibo.com/xxxxx", "", postMap)
*/
func (g *GatherStruct) Post(URL, refererURL string, postMap map[string]string) (html, returnedURL string, err error) {
	return g.PostUtil(URL, refererURL, "", postMap)
}

/*
post方式获取数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postMap:指post过去的相关数据
例:
ga := NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postMap := make(map[string]string)
postMap["user"] = "ydg"
postMap["password"] = "abcdef"
html, returnedURL, err := ga.PostUtil("https://weibo.com/xxxxx", "",cookies, postMap)
*/
func (g *GatherStruct) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, returnedURL string, err error) {
	postValues := url.Values{}
	for k, v := range postMap {
		postValues.Set(k, v)
	}
	postDataStr := postValues.Encode()
	postDataBytes := []byte(postDataStr)
	postBytesReader := bytes.NewReader(postDataBytes)
	if contentType, _ := g.Headers["Content-Type"]; contentType == "" {
		g.Headers["Content-Type"] = "application/x-www-form-urlencoded; param=value"
	}
	req, err := g.newHttpRequest("POST", URL, refererURL, cookies, postBytesReader)
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

//POST二进制
func (g *GatherStruct) PostBytes(URL, refererURL, cookies string, postBytes []byte) (html, returnedURL string, err error) {
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
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postXML:指待Post的XML数据，文本类型
例:
ga := gather.NewGather("chrome", false)
postXML := `<?xml version="1.0" encoding="utf-8"?><loin><user>ydg</user><passord>abcdef</passord></loin>`
html, returnedURL, err := ga.PostXML(`https://weibo.com/xxxxx`, "", postXML)
*/
func (g *GatherStruct) PostXML(URL, refererURL, postXML string) (html, returnedURL string, err error) {
	return g.PostXMLUtil(URL, refererURL, "", postXML)
}

/*
以XML的方式post数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
postXML:指待Post的XML数据，文本类型

例:
ga := gather.NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postXML := `<?xml version="1.0" encoding="utf-8"?><loin><user>ydg</user><passord>abcdef</passord></loin>`
html, returnedURL, err := ga.PostXML(`https://weibo.com/xxxxx`, "", cookies, postXML)
*/
func (g *GatherStruct) PostXMLUtil(URL, refererURL, cookies, postXML string) (html, returnedURL string, err error) {
	g.Headers["Content-Type"] = "application/xml"
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
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
postJson:指待Post的json数据，文本类型

例:
ga := gather.NewGather("chrome", false)
postJson := `{"user":"ydg","password":"abcdesg"}`
html, returnedURL, err := ga.PostJson(`https://weibo.com/xxxxx`, "", postJson)
*/
func (g *GatherStruct) PostJson(URL, refererURL, postJson string) (html, returnedURL string, err error) {
	return g.PostJsonUtil(URL, refererURL, "", postJson)
}

/*
以json的方式post数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
returnedURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
postJson:指待Post的json数据，文本类型

例:
ga := gather.NewGather("chrome", false)
cookies := `SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
postJson := `{"user":"ydg","password":"abcdesg"}`
html, returnedURL, err := ga.PostJsonUtil(`https://weibo.com/xxxxx`, "", cookies, postJson)
*/
func (g *GatherStruct) PostJsonUtil(URL, refererURL, cookies, postJson string) (html, returnedURL string, err error) {
	g.Headers["Content-Type"] = "application/json"
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
func (g *GatherStruct) PostMultipartformData(URL, refererURL, cookies, boundary string, postValueMap map[string]string, postFileMap map[string]multipartPostFile) (html, returnedURL string, err error) {
	return g.PostMultipartformDataUtil(URL, refererURL, "", boundary, postValueMap, postFileMap)
}

//multipart/form-data方式POST数据,自动继承先前的cookies
//boundary指post“分割边界”,这个“边界数据”不能在内容其他地方出现,一般来说使用一段从概率上说“几乎不可能”的数据即可
//postValueMap指post的普通文本,只包含name和value
//postFileMap指上传的文件,比如图片,需在调用此函数前自行转换成[]byte,当然POST协议也可使用base64编码后,不过在此忽略此用法,base64也请转换成[]byte
//multipart/form-data数据格式参见标准库中： mime\multipart\testdata\nested-mime,注意此处file文件是用的base64编码后的
func (g *GatherStruct) PostMultipartformDataUtil(URL, refererURL, cookies, boundary string, postValueMap map[string]string, postFileMap map[string]multipartPostFile) (html, returnedURL string, err error) {
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

	g.Headers["Content-Type"] = "multipart/form-data; boundary=" + boundary
	req, err := http.NewRequest("POST", URL, strings.NewReader(postData))
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}

//解压GZIP文件
func Ungzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err = ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil

}

//一个新的request对象
func (g *GatherStruct) newHttpRequest(method, URL, refererURL, cookies string, body io.Reader) (*http.Request, error) {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Sprintf("采集器可能未成功初始化,请先使用NewGather或NewGatherUtil或NewGatherProxy函数初始化再使用,具体错误信息:%v", err))
		}
	}()

	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return req, err
	}
	// Referer
	if refererURL != "" {
		g.Headers["Referer"] = refererURL
	}
	//cookies
	if cookies != "" {
		g.Headers["Cookie"] = cookies
	}
	//把header 按顺序添加进去
	type headerStruct struct {
		k string
		v string
	}
	var h []headerStruct
	for k, v := range g.Headers {
		h = append(h, headerStruct{k, v})
	}
	sort.Slice(h, func(i, j int) bool {
		return h[i].k <= h[j].k
	})
	for _, v := range h {
		req.Header.Set(v.k, v.v)
	}
	return req, nil
}

//最终抓取HTML
func (g *GatherStruct) request(req *http.Request) (html, returnedURL string, err error) {
	resp, err := g.Client.Do(req)

	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	// 200表示成功获取
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("http状态码:" + strconv.Itoa(resp.StatusCode))
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	//自动处理GZIP压缩的情况
	html, err = Ungzip(data)
	if err != nil {
		html = string(data)
	}
	return html, resp.Request.URL.String(), nil
}
