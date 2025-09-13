/*
 * Copyright (C) 2025 Arseniy Astankov
 *
 * This file is part of proxyflow.
 *
 * proxyflow is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
 *
 * proxyflow is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with proxyflow. If not, see <https://www.gnu.org/licenses/>.
 */
package proxy

import (
	"math/rand"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/etidart/proxyflow/internal/constants"
	"github.com/etidart/proxyflow/internal/logging"
)

type ProxyManager struct {
	cond sync.Cond
	proxies map[*Proxy]proxyStats
	badProxies map[*Proxy]proxyStats
	sortedProxies []*Proxy
}

// NewProxyManager initializes a new ProxyManager
func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies: make(map[*Proxy]proxyStats),
		badProxies: make(map[*Proxy]proxyStats),
		cond: *sync.NewCond(&sync.Mutex{}),
	}
}

type Message struct {
	Prx *Proxy
	Err string
	Dur time.Duration
}

// serves as channel receiving machine
func (pm *ProxyManager) ServeProxies(requests <-chan chan Message) {
	for {
		req := <-requests
		prx := pm.getBestProxy()
		req <- Message{Prx: prx}
		
		go func () {
			ans := <-req
			if ans.Err != "" {
				pm.addError(ans.Prx, ans.Err)
			} else if ans.Dur != 0 {
				pm.changeHandshakeAvg(ans.Prx, ans.Dur)
			}
		}()
	}
}

// same as ServeProxies() but gives not the best, but random proxy
func (pm *ProxyManager) ServeChecker(requests <-chan chan Message) {
	alreadyChecking := make(map[*Proxy]time.Time)
	for {
		pm.cond.L.Lock()
		for (len(pm.sortedProxies) == 0) {
			pm.cond.Wait()
		}
		proxyCopy := make([]*Proxy, len(pm.sortedProxies))
		copy(proxyCopy, pm.sortedProxies)
		pm.cond.L.Unlock()

		rand.Shuffle(len(proxyCopy), func(i, j int) {
			proxyCopy[i], proxyCopy[j] = proxyCopy[j], proxyCopy[i]
		})

		// for further deletion from alreadyChecking
		lookup := make(map[*Proxy]struct{})

		for _, proxy := range proxyCopy {
			lookup[proxy] = struct{}{}

			req := <-requests
			tval, isIn := alreadyChecking[proxy]
			var good bool
			if !isIn {
				alreadyChecking[proxy] = time.Now()
				good = true
			} else {
				if time.Since(tval) < constants.PRXCHKCD {
					good = false
				} else {
					alreadyChecking[proxy] = time.Now()
					good = true
				}
			}

			if good {
				req <- Message{Prx: proxy}
				go func() {
					ans := <-req
					if ans.Err != "" {
						pm.addError(ans.Prx, ans.Err)
					} else if ans.Dur != 0 {
						pm.changeHandshakeAvg(ans.Prx, ans.Dur)
					}
				}()
			}
		}

		// cleanup alreadyChecking using lookup
		for key := range alreadyChecking {
			if _, found := lookup[key]; !found {
				delete(alreadyChecking, key)
			}
		}

		// optimize by sleeping
		var earlistTime time.Time
		for _, t := range alreadyChecking {
			if earlistTime.IsZero() || t.Before(earlistTime) {
				earlistTime = t
			}
		}
		time.Sleep(constants.PRXCHKCD - time.Since(earlistTime))
	}
}

// appends a proxy to manager with specified handshakeAvg
func (pm *ProxyManager) addProxyHS(addr string, prot Protocol, hsavg time.Duration) {
	pm.cond.L.Lock()
	defer pm.cond.L.Unlock()
	proxy := &Proxy{
		Address: addr,
		Proto: prot,
	}
	pm.proxies[proxy] = proxyStats{
		handshakeAvg: hsavg,
		errors: 0,
	}
	pm.sortedProxies = append(pm.sortedProxies, proxy)
	pm.sortProxies()
}

// appends a proxy to manager
func (pm *ProxyManager) AddProxy(addr string, prot Protocol) {
	pm.addProxyHS(addr, prot, constants.PRXDEFHSAVG)
}

// update the handshakeAvg
func (pm *ProxyManager) changeHandshakeAvg(prx *Proxy, newVal time.Duration) {
	pm.cond.L.Lock()
	defer pm.cond.L.Unlock()

	stats, exists := pm.proxies[prx]
	if !exists {
		//logging.Warn("changeHandshakeAvg: proxy not found")
		return
	}

	var difference time.Duration
	if newVal >= stats.handshakeAvg {
		difference = newVal - stats.handshakeAvg
	} else {
		difference = stats.handshakeAvg - newVal
	}
	if difference >= constants.PRXMINHSAVGDIFF {
		stats.handshakeAvg = (stats.handshakeAvg + newVal) / 2
		pm.proxies[prx] = stats
		pm.sortProxies()
	}
}

// increment errors counter (del when limit is reached)
func (pm *ProxyManager) addError(prx *Proxy, error string) {
	if strings.HasPrefix(error, "crit") {
		pm.cond.L.Lock()
		defer pm.cond.L.Unlock()

		stats, exists := pm.proxies[prx]
		if !exists {
			//logging.Warn("addError: proxy not found")
			return
		}
		stats.errors++
		stats.lastErr = error
		pm.proxies[prx] = stats
		if stats.errors > constants.PRXMAXERRS {
			delete(pm.proxies, prx)
			pm.badProxies[prx] = stats
			pm.rmFromSorted(prx)
			logging.Warn("proxy " + prx.Address + " is removed due to exceeding the error limit (last err \"" + error +"\")")
		}
	}
}

// returns the best available proxy
func (pm *ProxyManager) getBestProxy() *Proxy {
	pm.cond.L.Lock()
	defer pm.cond.L.Unlock()
	if len(pm.sortedProxies) == 0 {
		// rotate maps
		for k, v := range pm.badProxies {
			pm.addProxyHS(k.Address, k.Proto, v.handshakeAvg)
		}
		pm.badProxies = make(map[*Proxy]proxyStats)
		pm.cond.Broadcast()
	}
	return pm.sortedProxies[0]
}

func (pm *ProxyManager) sortProxies() {
	sort.Slice(pm.sortedProxies, func(i, j int) bool {
		return pm.proxies[pm.sortedProxies[i]].handshakeAvg < pm.proxies[pm.sortedProxies[j]].handshakeAvg
	})
}

func (pm *ProxyManager) rmFromSorted(proxy *Proxy) {
	for i, p := range pm.sortedProxies {
		if p == proxy {
			pm.sortedProxies = slices.Delete(pm.sortedProxies, i, i+1)
			break
		}
	}
}