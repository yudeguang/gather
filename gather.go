package files

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

//客户端类型 chrome ,ie,firefox,
var Agent string

//采集类
var gather Gather

type Gather struct {
	client *http.Client
}

//默认传一个nil Jar进去
func New() *Gather {
	j := newWebCookieJar()
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	gather.client = &http.Client{Transport: tr, Jar: j}
	return &gather
}

//一个新的request对象，里面先设置好浏览器那些
func newHttpRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return req, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	switch strings.ToLower(Agent) {
	case "baidu":
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Baiduspider/2.0;++http://www.baidu.com/search/spider.html)")
	case "google":
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1;+http://www.google.com/bot.html)")
	case "bing":
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bingbot/2.0;+http://www.bing.com/bingbot.htm)")
	case "chrome":
		req.Header.Set("key", `Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.94 Safari/537.36`)
	default:
		req.Header.Set("key", `Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.94 Safari/537.36`)
	}
	return req, nil
}

//GET方式获取数据,手动设置Cookie
func (this *Gather) getUtil(URL, refererURL, cookies string) (html string, returnedURL string, status int) {

	req, err := newHttpRequest("GET", URL, nil)
	hasErrFatal(err)
	//有时需要加Referer参数
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	resp, err := this.client.Do(req)
	hasErrFatal(err)
	defer resp.Body.Close()
	// 200表示成功获取
	if resp.StatusCode != 200 {
		log.Println(resp.StatusCode)
		return "", "", 0
	}
	data, err := ioutil.ReadAll(resp.Body)
	if hasErrPrintln(err) {
		return "", "", 0
	}
	//下面很可能还有存在GZIP压缩的情况
	return string(data), resp.Request.URL.String(), 1
}

//post 方式获取数据 手动设置Cookie
func (this *Gather) postUtil(URL, refererURL, cookies string, post map[string]string) (html string, url2 string, status int) {
	postValues := url.Values{}
	for k, v := range post {
		postValues.Set(k, v)
	}
	postDataStr := postValues.Encode()
	postDataBytes := []byte(postDataStr)
	postBytesReader := bytes.NewReader(postDataBytes)
	req, err := http.NewRequest("POST", URL, postBytesReader)
	hasErrFatal(err)
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
	hasErrFatal(err)

	defer resp.Body.Close()
	// 判断是否读取成功 200为成功标识
	if resp.StatusCode != 200 {
		log.Println(resp.StatusCode)
		return "", "", 0
	}
	data, err := ioutil.ReadAll(resp.Body)
	if hasErrPrintln(err) {
		return "", "", 0
	}
	//下面很可能还有存在GZIP压缩的情况
	return string(data), resp.Request.URL.String(), 1
}

//GET方式获取数据,手动设置Cookie
func (this *Gather) get(URL, refererURL string) (html string, returnedURL string, status int) {
	return this.getUtil(URL, refererURL, "")
}

//post方式获取数据,手动设置Cookie
func (this *Gather) post(URL, refererURL string, post map[string]string) (html string, url2 string, status int) {
	return this.postUtil(URL, refererURL, "", post)
}
