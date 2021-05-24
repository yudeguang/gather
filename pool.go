package gather

import (
	"fmt"
	"sync"
	"time"
)

type Pool struct {
	unUsed sync.Map        //空闲的Pool下标
	pool   []*GatherStruct //缓存池
	locker sync.Mutex
}

var errNoFreeClinetFind = fmt.Errorf("time out,no free client find")

//池化技术 同时申明若干个，以备使用，避免频繁的申明回收,最多100个
func NewGatherUtilPool(headers map[string]string, proxyURL string, timeOut int, isCookieLogOpen bool, num int) *Pool {
	if num <= 0 {
		num = 1
	}
	if num > 100 {
		num = 100
	}
	//重设一下maxIdleConns以适应不同的需求
	if num >= 10 && num <= 100 {
		maxIdleConns = num
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
	pool_index := p.getPoolIndex()
	if pool_index == -1 {
		return "", "", errNoFreeClinetFind
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].Get(URL, refererURL)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) GetUtil(URL, refererURL, cookies string) (html, redirectURL string, err error) {
	pool_index := p.getPoolIndex()
	if pool_index == -1 {
		return "", "", errNoFreeClinetFind
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].GetUtil(URL, refererURL, cookies)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) Post(URL, refererURL string, postMap map[string]string) (html, redirectURL string, err error) {
	pool_index := p.getPoolIndex()
	if pool_index == -1 {
		return "", "", errNoFreeClinetFind
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].Post(URL, refererURL, postMap)
}

//从缓存池中 随便获取一个，然后再利用
func (p *Pool) PostUtil(URL, refererURL, cookies string, postMap map[string]string) (html, redirectURL string, err error) {
	pool_index := p.getPoolIndex()
	if pool_index == -1 {
		return "", "", errNoFreeClinetFind
	}
	defer p.unUsed.Store(pool_index, true)
	return p.pool[pool_index].PostUtil(URL, refererURL, cookies, postMap)
}

//设置超时30秒超时 ，如果没有找到就返回-1表示失败
func (p *Pool) getPoolIndex() int {
	p.locker.Lock()
	defer p.locker.Unlock()
	pool_index := -1
	for num := 0; num < 600; num++ {
		p.unUsed.Range(func(k, v interface{}) bool {
			pool_index = k.(int)
			if pool_index == -1 {
				return true
			} else {
				//false表示不再继续遍历
				return false
			}
		})
		if pool_index != -1 {
			p.unUsed.Delete(pool_index)
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
	return pool_index
}
