// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package gather

import (
	"log"
	"net/http"
	"net/url"
	"sync"
)

//cookie的保存对象
type webCookieJar struct {
	lk            sync.Mutex
	cookies       map[string][]*http.Cookie
	cookieLogOpen bool
}

func newWebCookieJar(isCookieLogOpen bool) *webCookieJar {
	jar := new(webCookieJar)
	jar.cookieLogOpen = isCookieLogOpen
	jar.cookies = make(map[string][]*http.Cookie)
	return jar
}
func (j *webCookieJar) SetCookies(u *url.URL, newCookies []*http.Cookie) {
	j.lk.Lock()
	defer j.lk.Unlock()
	//如果原来有了就覆盖,根据host和Path判断
	oldCookies := j.cookies[u.Host]
	if j.cookieLogOpen {
		log.Println("COOKIE变更:", u.String())
	}
	for newIndex := 0; newIndex < len(newCookies); newIndex++ {
		isFound := false
		for oldIndex := 0; oldIndex < len(oldCookies); oldIndex++ {
			if oldCookies[oldIndex].Name == newCookies[newIndex].Name &&
				oldCookies[oldIndex].Path == newCookies[newIndex].Path {
				//原来有的，就直接替换就可以
				oldCookies[oldIndex] = newCookies[newIndex]
				if j.cookieLogOpen {
					log.Println("替换cookie:", newCookies[newIndex].String())
				}
				isFound = true
				break
			}
		}
		if !isFound {
			oldCookies = append(oldCookies, newCookies[newIndex])
			if j.cookieLogOpen {
				log.Println("添加cookie:", newCookies[newIndex].String())
			}
		}
	}
	j.cookies[u.Host] = oldCookies
}
func (j *webCookieJar) Cookies(u *url.URL) []*http.Cookie {
	cookies := j.cookies[u.Host]
	/*log.Println("URL:", u.String())
	for i, c := range cookies {
		log.Println("cookie:", i, c.String())
	}*/
	return cookies
}
