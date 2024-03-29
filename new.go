// Copyright 2020 ratelimit Author(https://github.com/yudeguang/gather). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/gather.
//模拟浏览器进行数据采集包,可较方便的定义http头，同时全自动化处理cookies
package gather

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

//MaxIdleConns，与idle池有关,不同于MaxIdleConnsPerHost只针对某个host，MaxIdleConns是针对整
//个Client的所有idle池中的连接数的和，这个和不能超过MaxIdleConns
//这里，因为我们每个Client实际只连接一台主机，所以让MaxIdleConns与MaxIdleConnsPerHost一致
var maxIdleConns = 100

//内部变量全部大写导出，允许在执行过程中任意修改
type GatherStruct struct {
	Client      *http.Client
	Headers     map[string]string
	safeHeaders sync.Map
	J           *webCookieJar
	//有较小的概率，如果多人都是用的同一个对象抓取，会出现 fatal error: concurrent map writes
	//所以，建议是每个程序创建单独对象
	locker sync.Mutex
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
	gather.Client = &http.Client{Transport: getHttpTransport(proxyURL), Jar: gather.J}
	gather.Client.Timeout = time.Duration(timeOut) * time.Second
	for k, v := range gather.Headers {
		gather.safeHeaders.Store(k, v)
	}
	return &gather
}

//全局用特定的httpTransport,只有无代理时，可以复用
var transportNoProxy *http.Transport = nil

var transportLocker sync.Mutex

func getHttpTransport(proxyURL string) *http.Transport {
	transportLocker.Lock()
	defer transportLocker.Unlock()
	if proxyURL == "" {
		if transportNoProxy == nil {
			transportNoProxy = &http.Transport{
				//DisableKeepAlives:  true, //默认值为false，即启动keep-alive。若将其置为false，则关闭keep-alive
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: true}, //忽略认证
				DisableCompression: true,
				Dial: func(netw, addr string) (net.Conn, error) {
					c, err := net.DialTimeout(netw, addr, time.Second*10)
					if err != nil {
						return nil, err
					}
					c.(*net.TCPConn).SetLinger(3)
					return c, nil
				},
				//copy from http.DefaultTransport
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          maxIdleConns,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
			// 默认值为2，单机最大允许连接数，即长连接,默认值太小,这里，因为我们的实际应用往往是一个client只连接一台主机，所以直接设成与maxIdleConns相等
			transportNoProxy.MaxIdleConnsPerHost = maxIdleConns
		}
		return transportNoProxy
	} else {
		//使用代理时，不能复用，因为代理一般需要经常更换
		var transportWithProxy *http.Transport = nil
		//设置代理服务器 proxyUrl 指类似 https://104.207.139.207:8080 http://104.207.139.207:8080
		if transportWithProxy == nil {
			proxy := func(_ *http.Request) (*url.URL, error) { return url.Parse(proxyURL) }
			transportWithProxy = &http.Transport{
				//DisableKeepAlives:  true, //默认值为false，即启动keep-alive。若将其置为false，则关闭keep-alive
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
				DisableCompression: true,
				Proxy:              proxy,
				Dial: func(netw, addr string) (net.Conn, error) {
					c, err := net.DialTimeout(netw, addr, time.Second*10)
					if err != nil {
						return nil, err
					}
					c.(*net.TCPConn).SetLinger(3)
					return c, nil
				},
				//copy from http.DefaultTransport
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          maxIdleConns,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
			// 默认值为2，单机最大允许连接数，即长连接,默认值太小,这里，因为我们的实际应用往往是一个client只连接一台主机，所以直接设成与maxIdleConns相等
			transportWithProxy.MaxIdleConnsPerHost = maxIdleConns
		}
		return transportWithProxy
	}
}
