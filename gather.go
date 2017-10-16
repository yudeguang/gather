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
	"strconv"
	"strings"
	"time"
)

/*
数据采集类，可自动处理Cookie,共包含4种方法，其中get,post为自动处理cookie
GetUtil与PostUtil为强行设置cookie，这种情况一般是用于登录有验证码时，手工
把验证码设置进去。
*/

type GatherStruct struct {
	client    *http.Client
	headerMap map[string]string //定制header
	agent     string            //声明客户端名称
	J         *webCookieJar     //webCookieJar可以导出，自由修改
}

//multipart/form-data 上传文件的结构体
type postFile struct {
	fileName    string
	contentType string
	content     []byte
}

//实例化Gather，HTTP头中除Agent外，其它全部默认, isCookieLogOpen为Cookie变更时是否打印
func NewGather(defaultAgent string, isCookieLogOpen bool) *GatherStruct {
	var gather GatherStruct
	gather.agent = defaultAgent
	gather.J = newWebCookieJar(isCookieLogOpen)
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	gather.client = &http.Client{Transport: tr, Jar: gather.J}
	//设置超时，默认300秒
	gather.client.Timeout = 300 * time.Second
	return &gather
}

//实例化Gather，HTTP头可以全部自定义,isCookieLogOpen为Cookie变更时是否打印
func NewCustomizedGather(headerMap map[string]string, isCookieLogOpen bool) *GatherStruct {
	var gather GatherStruct
	gather.headerMap = headerMap
	gather.J = newWebCookieJar(isCookieLogOpen)
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	gather.client = &http.Client{Transport: tr, Jar: gather.J}
	//设置超时，默认300秒
	gather.client.Timeout = 300 * time.Second
	return &gather
}

//一个新的request对象，里面先设置好浏览器那些
func (this *GatherStruct) newHttpRequest(method, URL string, body io.Reader) (*http.Request, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal("采集器可能未初始化,请先使用NewGather或NewCustomizedGather函数初始化", r)
		}
	}()

	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		log.Println(err)
		return req, err
	}
	//Host
	temp, err := url.Parse(URL)
	if err != nil {
		return req, err
	}
	req.Header.Set("Host", temp.Host)
	//两种情况，
	if len(this.headerMap) == 0 && len(this.agent) != 0 {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Encoding", "gzip, deflate, sdch")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		switch strings.ToLower(this.agent) {
		case "baidu":
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Baiduspider/2.0;++http://www.baidu.com/search/spider.html)")
		case "google":
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1;+http://www.google.com/bot.html)")
		case "bing":
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bingbot/2.0;+http://www.bing.com/bingbot.htm)")
		case "chrome":
			req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36`)
		case "360":
			req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/45.0.2454.101 Safari/537.36`)
		case "ie", "ie9":
			req.Header.Set("User-Agent", `Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Win64; x64; Trident/5.0)`)
		case "":
			req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.94 Safari/537.36`)
		default:
			req.Header.Set("User-Agent", this.agent)
		}
	} else if len(this.headerMap) != 1 && len(this.agent) == 0 {
		for k, v := range this.headerMap {
			req.Header.Set(k, v)
		}
	} else {
		return req, fmt.Errorf("agent未定义")
	}

	return req, nil
}

//GET方式获取数据,手动设置Cookie
func (this *GatherStruct) GetUtil(URL, refererURL, cookies string) (html, returnedURL string, err error) {
	req, err := this.newHttpRequest("GET", URL, nil)
	if err != nil {
		return "", "", err
	}
	//有时需要加Referer参数
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
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

//post 方式获取数据 手动设置Cookie
func (this *GatherStruct) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, returnedURL string, err error) {
	postValues := url.Values{}
	for k, v := range postMap {
		postValues.Set(k, v)
	}
	postDataStr := postValues.Encode()
	postDataBytes := []byte(postDataStr)
	postBytesReader := bytes.NewReader(postDataBytes)
	req, err := this.newHttpRequest("POST", URL, postBytesReader)
	if err != nil {
		return "", "", err
	}
	//post特有HEADER信息,这个一定要加，不加form的值post不过去
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	//有时需要加Referer参数
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	resp, err := this.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	// 判断是否读取成功 200为成功标识
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

//以XML的方式post数据
func (this *GatherStruct) PostXML(URL, refererURL, cookies string, postXML string) (html, returnedURL string, err error) {
	req, err := this.newHttpRequest("POST", URL, strings.NewReader(postXML))
	if err != nil {
		return "", "", err
	}
	//post特有HEADER信息,这个一定要加，不加form的值post不过去
	req.Header.Set("Content-Type", "application/xml")
	//有时需要加Referer参数
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	resp, err := this.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	// 判断是否读取成功 200为成功标识
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

//multipart/form-data方式POST数据
//boundary指post“分割边界”，这个“边界数据”不能在内容其他地方出现，一般来说使用一段从概率上说“几乎不可能”的数据即可
//postValueMap指post的普通文本，只包含name和value
//postFileMap指上传的文件，比如图片，需在调用此函数前自行转换成[]byte，当然POST协议也可使用base64编码后，不过在此忽略此用法，base64也请转换成[]byte
//multipart/form-data数据格式参见标准库中： mime\multipart\testdata\nested-mime,注意此处file文件是用的base64编码后的
func (this *GatherStruct) PostMultipartformData(URL, refererURL, cookies string, boundary string, postValueMap map[string]string, postFileMap map[string]postFile) (html, returnedURL string, err error) {
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

	req, err := http.NewRequest("POST", URL, strings.NewReader(postData))
	if err != nil {
		return "", "", err
	}
	//post特有HEADER信息,这个一定要加，不加form的值post不过去
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	//有时需要加Referer参数
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	resp, err := this.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	// 判断是否读取成功 200为成功标识
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
