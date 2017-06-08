package gather

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

/*
数据采集类，可自动处理Cookie,共包含4种方法，其中get,post为自动处理cookie
GetUtil与PostUtil为强行设置cookie，这种情况一般是用于登录有验证码时，手工
把验证码设置进去。
*/
type GatherStruct struct {
	client *http.Client
	agent  string
}

//实例化Gather，defaultAgent为默认客户端, isCookieLogOpen为Cookie变更时是否打印
func NewGather(defaultAgent string, isCookieLogOpen bool) *GatherStruct {
	var gather GatherStruct
	gather.agent = defaultAgent
	j := newWebCookieJar(isCookieLogOpen)
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	gather.client = &http.Client{Transport: tr, Jar: j}
	return &gather
}

//一个新的request对象，里面先设置好浏览器那些
func (this *GatherStruct) newHttpRequest(method, URL string, body io.Reader) (*http.Request, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal("采集器可能未初始化,请先初始化再使用(使用NewGather函数初始化) ", r)
		}
	}()
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		log.Println(err)
		return req, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, sdch")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	temp, err := url.Parse(URL)
	if err != nil {
		return req, err
	}
	req.Header.Set("Host", temp.Host)
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

	default:
		req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.94 Safari/537.36`)
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
	//下面很可能还有存在GZIP压缩的情况
	return string(data), resp.Request.URL.String(), nil
}

//post 方式获取数据 手动设置Cookie
func (this *GatherStruct) PostUtil(URL, refererURL, cookies string, post map[string]string) (html, returnedURL string, err error) {
	postValues := url.Values{}
	for k, v := range post {
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
	//其实data还可能是gzip压缩后的结果
	return string(data), resp.Request.URL.String(), nil
}

//GET方式获取数据,手动设置Cookie
func (this *GatherStruct) Get(URL, refererURL string) (html, returnedURL string, err error) {
	return this.GetUtil(URL, refererURL, "")
}

//post方式获取数据,手动设置Cookie
func (this *GatherStruct) Post(URL, refererURL string, post map[string]string) (html, returnedURL string, err error) {
	return this.PostUtil(URL, refererURL, "", post)
}
