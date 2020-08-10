// Copyright 2020 ratelimit Author(https://github.com/yudeguang/gather). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/gather.
//模拟浏览器进行数据采集包,可较方便的定义http头，同时全自动化处理cookies
package gather

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
)

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
	//注意200,202都表示成功
	if !(resp.StatusCode == 200 || resp.StatusCode == 202) {
		return "", "", fmt.Errorf("http状态码:" + strconv.Itoa(resp.StatusCode))
	}
	var data []byte
	// if g.HTMLShouldConvertToUTF8 {
	// 	//判断网页是什么编码
	// 	e := determineEncoding(resp.Body)
	// 	//转换为utf8
	// 	utf8Reader := transform.NewReader(resp.Body, e.NewDecoder())
	// 	data, err = ioutil.ReadAll(utf8Reader)
	// } else {
	data, err = ioutil.ReadAll(resp.Body)
	//}

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
