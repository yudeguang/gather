package gather

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

/*
数据采集类,可自动处理Cookie,共包含4种方法,其中get,post为自动处理cookie
GetUtil与PostUtil为强行设置cookie,这种情况一般是用于登录有验证码时,手工
把验证码设置进去。
*/

//Headers与J(cookies)都导出，允许在执行过程中任意修改
type GatherStruct struct {
	client  *http.Client
	Headers map[string]string //定制header
	J       *webCookieJar     //cookies
}

//multipart/form-data 上传文件的结构体
type multipartPostFile struct {
	fileName    string
	contentType string
	content     []byte
}

//实例化Gather,HTTP头中除Agent外,其它全部默认, isCookieLogOpen为Cookie变更时是否打印
func NewGather(defaultAgent string, isCookieLogOpen bool) *GatherStruct {
	var headers = make(map[string]string)
	headers["User-Agent"] = defaultAgent
	return NewGatherUtil(headers, 300, isCookieLogOpen)
}

//实例化Gather,HTTP头可以全部自定义,isCookieLogOpen为Cookie变更时是否打印
func NewGatherUtil(headers map[string]string, timeOut int, isCookieLogOpen bool) *GatherStruct {
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
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	gather.client = &http.Client{Transport: tr, Jar: gather.J}
	gather.client.Timeout = time.Duration(timeOut) * time.Second
	return &gather
}

//一个新的request对象
func (this *GatherStruct) newHttpRequest(method, URL, refererURL, cookies string, body io.Reader) (*http.Request, error) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Fatal("采集器可能未初始化,请先使用NewGather或NewCustomizedGather函数初始化", r)
	// 	}
	// }()

	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		log.Println(err)
		return req, err
	}
	//Host 奇怪的是，client会默认加上HOST属性，并且永远放在请求的前部
	urlInfo, err := url.Parse(URL)
	if err != nil {
		return req, err
	}
	this.Headers["Host"] = urlInfo.Host
	// Referer
	if refererURL != "" {
		this.Headers["Referer"] = refererURL
	}
	//cookies
	if cookies != "" {
		this.Headers["Cookie"] = cookies
	}
	//把header 按顺序添加进去
	type headerStruct struct {
		k string
		v string
	}
	var h []headerStruct
	for k, v := range this.Headers {
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
func (this *GatherStruct) request(req *http.Request) (html, returnedURL string, err error) {
	resp, err := this.client.Do(req)
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
	//处理GZIP压缩的情况
	html, err = Ungzip(data)
	if err != nil {
		html = string(data)
	}
	return html, resp.Request.URL.String(), nil
}

//GET方式获取数据,手动设置Cookie
func (this *GatherStruct) GetUtil(URL, refererURL, cookies string) (html, returnedURL string, err error) {
	req, err := this.newHttpRequest("GET", URL, refererURL, cookies, nil)
	if err != nil {
		return "", "", err
	}
	return this.request(req)
}

//post 方式获取数据 手动设置Cookie
func (this *GatherStruct) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, returnedURL string, err error) {
	postValues := url.Values{}
	for k, v := range postMap {
		postValues.Set(k, v)
	}
	postDataStr := postValues.Encode()
	postDataBytes := []byte(postDataStr)
	postBytesReader := bytes.NewReader(postDataBytes)
	//post Content-Type
	this.Headers["Content-Type"] = "application/x-www-form-urlencoded; param=value"
	req, err := this.newHttpRequest("POST", URL, refererURL, cookies, postBytesReader)
	if err != nil {
		return "", "", err
	}
	return this.request(req)
}

//以XML的方式post数据
func (this *GatherStruct) PostXML(URL, refererURL, cookies string, postXML string) (html, returnedURL string, err error) {
	this.Headers["Content-Type"] = "application/xml"
	req, err := this.newHttpRequest("POST", URL, refererURL, cookies, strings.NewReader(postXML))
	if err != nil {
		return "", "", err
	}
	return this.request(req)
}

//multipart/form-data方式POST数据
//boundary指post“分割边界”,这个“边界数据”不能在内容其他地方出现,一般来说使用一段从概率上说“几乎不可能”的数据即可
//postValueMap指post的普通文本,只包含name和value
//postFileMap指上传的文件,比如图片,需在调用此函数前自行转换成[]byte,当然POST协议也可使用base64编码后,不过在此忽略此用法,base64也请转换成[]byte
//multipart/form-data数据格式参见标准库中： mime\multipart\testdata\nested-mime,注意此处file文件是用的base64编码后的
func (this *GatherStruct) PostMultipartformData(URL, refererURL, cookies string, boundary string, postValueMap map[string]string, postFileMap map[string]multipartPostFile) (html, returnedURL string, err error) {
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

	this.Headers["Content-Type"] = "multipart/form-data; boundary=" + boundary
	req, err := http.NewRequest("POST", URL, strings.NewReader(postData))
	if err != nil {
		return "", "", err
	}
	return this.request(req)
}

//GET方式获取数据,手动设置Cookie
func (this *GatherStruct) Get(URL, refererURL string) (html, returnedURL string, err error) {
	return this.GetUtil(URL, refererURL, "")
}

//post方式获取数据,手动设置Cookie
func (this *GatherStruct) Post(URL, refererURL string, post map[string]string) (html, returnedURL string, err error) {
	return this.PostUtil(URL, refererURL, "", post)
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
