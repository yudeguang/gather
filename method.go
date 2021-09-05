// Copyright 2020 ratelimit Author(https://github.com/yudeguang/gather). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/gather.
//模拟浏览器进行数据采集包,可较方便的定义http头，同时全自动化处理cookies
package gather

/*
GET方式获取数据,自动继承先前的cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL

例:
ga := NewGather("chrome", false)
html, redirectURL, err := ga.Method("OPTIONS","https://www.baidu.com/", "")
*/
func (g *GatherStruct) Method(method, URL, refererURL string) (html, redirectURL string, err error) {
	return g.MethodUtil(method, URL, refererURL, "")
}

/*
GET方式获取数据,手动增加cookies
URL:指待抓取的URL
refererURL:上一次访问的URL。某些防抓取比较严格的网站会对上次访问的页面URL进行验证
cookies:文本形式，对于某些要求登录的网站，登录之后，直接从浏览器中把Cookie复制进去即可
redirectURL:最终实际访问到内容的URL。因为有时候会碰到301跳转等情况，最终访问的URL并非输入的URL

例:
ga:= NewGather("chrome", false)
cookies:=`SINAGLOBAL=8868584542946.604.1509350660873;??????????; YF-Page-G0=b9385a03a044baf8db46b84f3ff125a0`
html, redirectURL, err := ga.MethodUtil("OPTIONS","https://weibo.com/xxxxxx",cookies, "")
*/
//GET方式获取数据,手动设置Cookie,Cookie留空则自动继承上次抓取时使用的Cookie
func (g *GatherStruct) MethodUtil(method, URL, refererURL, cookies string) (html, redirectURL string, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	req, err := g.newHttpRequest(method, URL, refererURL, cookies, nil)
	if err != nil {
		return "", "", err
	}
	return g.request(req)
}
