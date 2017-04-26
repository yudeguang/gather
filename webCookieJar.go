package files

import (
	"log"
	"net/http"
	"net/url"
	"sync"
)

//是否打印COOKIE的变更
var logOn = false

//cookie的保存对象
type webCookieJar struct {
	lk      sync.Mutex
	cookies map[string][]*http.Cookie
}

func newWebCookieJar() *webCookieJar {
	jar := new(webCookieJar)
	jar.cookies = make(map[string][]*http.Cookie)
	return jar
}
func (j *webCookieJar) SetCookies(u *url.URL, newCookies []*http.Cookie) {
	j.lk.Lock()
	defer j.lk.Unlock()
	//如果原来有了就覆盖,根据host和Path判断
	oldCookies := j.cookies[u.Host]
	if logOn {
		log.Println("COOKIE变更:", u.String())
	}
	for newIndex := 0; newIndex < len(newCookies); newIndex++ {
		isFound := false
		for oldIndex := 0; oldIndex < len(oldCookies); oldIndex++ {
			if oldCookies[oldIndex].Name == newCookies[newIndex].Name &&
				oldCookies[oldIndex].Path == newCookies[newIndex].Path {
				//原来有的，就直接替换就可以
				oldCookies[oldIndex] = newCookies[newIndex]
				if logOn {
					log.Println("替换cookie:", newCookies[newIndex].String())
				}
				isFound = true
				break
			}
		}
		if !isFound {
			oldCookies = append(oldCookies, newCookies[newIndex])
			if logOn {
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
