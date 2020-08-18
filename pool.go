package gather

import (
	"sync"
)

type Pool struct {
	unUsed sync.Map        //空闲的Pool下标
	pool   []*GatherStruct //缓存池
}

//池化技术 同时申明若干个，以备使用，避免频繁的申明回收,最多100个
func NewGatherUtilPool(headers map[string]string, proxyURL string, timeOut int, isCookieLogOpen bool, num int) *Pool {
	if num <= 0 {
		num = 1
	}
	if num > 100 {
		num = 100
	}
	var gp Pool

	for i := 0; i < num; i++ {
		ga := NewGatherUtil(headers, proxyURL, timeOut, isCookieLogOpen)
		gp.pool = append(gp.pool, ga)
		gp.unUsed.Store(i, true)
	}
	return &gp
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) Get(URL, refererURL string) (html, redirectURL string, err error) {
	//这个地方要找到有能用的为止
	find := false
	pool_index := -1
	for {
		p.unUsed.Range(func(k, v interface{}) bool {
			if pool_index == -1 {
				pool_index = k.(int)
				find = true
			}
			return true
		})
		if find {
			p.unUsed.Delete(pool_index)
			break
		}
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].Get(URL, refererURL)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) GetUtil(URL, refererURL, cookies string) (html, redirectURL string, err error) {
	//这个地方要找到有能用的为止
	find := false
	pool_index := -1
	for {
		p.unUsed.Range(func(k, v interface{}) bool {
			if pool_index == -1 {
				pool_index = k.(int)
				find = true
			}
			return true
		})
		if find {
			p.unUsed.Delete(pool_index)
			break
		}
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].GetUtil(URL, refererURL, cookies)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) Post(URL, refererURL string, postMap map[string]string) (html, redirectURL string, err error) {
	//这个地方要找到有能用的为止
	find := false
	pool_index := -1
	for {
		p.unUsed.Range(func(k, v interface{}) bool {
			if pool_index == -1 {
				pool_index = k.(int)
				find = true
			}
			return true
		})
		if find {
			p.unUsed.Delete(pool_index)
			break
		}
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].Post(URL, refererURL, postMap)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, redirectURL string, err error) {
	//这个地方要找到有能用的为止
	find := false
	pool_index := -1
	for {
		p.unUsed.Range(func(k, v interface{}) bool {
			if pool_index == -1 {
				pool_index = k.(int)
				find = true
			}
			return true
		})
		if find {
			p.unUsed.Delete(pool_index)
			break
		}
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].PostUtil(URL, refererURL, cookies, postMap)
}
