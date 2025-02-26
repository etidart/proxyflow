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

	"github.com/etidart/proxyflow/internal/logging"
)

const MAXERRORS = 2

type ProxyManager struct {
	mu sync.Mutex
	proxies map[*Proxy]proxyStats
	badProxies map[*Proxy]proxyStats
	sortedProxies []*Proxy
}

// NewProxyManager initializes a new ProxyManager
func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies: make(map[*Proxy]proxyStats),
		badProxies: make(map[*Proxy]proxyStats),
	}
}

type Message struct {
	Prx *Proxy
	Err string
	Dur time.Duration
}

// serves as channel receiving machine (ha-ha)
func (pm *ProxyManager) ServeProxies(requests <-chan chan Message) {
	for {
		req := <-requests
		prx := pm.getBestProxy()
		req <- Message{Prx: prx}
		
		if prx != nil {
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
}

// same as ServeProxies() but gives not the best, but random proxy
func (pm *ProxyManager) ServeChecker(requests <-chan chan Message) {
	for {
		pm.mu.Lock()
		if len(pm.sortedProxies) == 0 {
			logging.Error("ServeChecker: no available proxies, but required to work")
			pm.mu.Unlock()
			time.Sleep(time.Duration(100) * time.Millisecond)
			continue
		}
		proxyCopy := make([]*Proxy, len(pm.sortedProxies))
		copy(proxyCopy, pm.sortedProxies)
		pm.mu.Unlock()

		rand.Shuffle(len(proxyCopy), func(i, j int) {
			proxyCopy[i], proxyCopy[j] = proxyCopy[j], proxyCopy[i]
		})

		for _, proxy := range proxyCopy {
			req := <-requests
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
}

// adds a new proxy to manager
func (pm *ProxyManager) AddProxy(addr string, prot Protocol) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	proxy := &Proxy{
		Address: addr,
		Proto: prot,
	}
	pm.proxies[proxy] = proxyStats{
		handshakeAvg: 0,
		errors: 0,
	}
	pm.sortedProxies = append(pm.sortedProxies, proxy)
	pm.sortProxies()
}

// update the handshakeAvg
func (pm *ProxyManager) changeHandshakeAvg(prx *Proxy, newVal time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	stats, exists := pm.proxies[prx]
	if !exists {
		logging.Warn("changeHandshakeAvg: proxy not found")
		return
	}
	if (newVal > stats.handshakeAvg && newVal - stats.handshakeAvg >= time.Duration(500) * time.Millisecond) || newVal < stats.handshakeAvg {
		stats.handshakeAvg = (stats.handshakeAvg + newVal) / 2
		pm.proxies[prx] = stats
		pm.sortProxies()
	}
}

// increment errors counter (del on 3rd error)
func (pm *ProxyManager) addError(prx *Proxy, error string) {
	if strings.HasPrefix(error, "crit") {
		pm.mu.Lock()
		defer pm.mu.Unlock()

		stats, exists := pm.proxies[prx]
		if !exists {
			logging.Warn("addError: proxy not found")
			return
		}
		stats.errors++
		stats.lastErr = error
		pm.proxies[prx] = stats
		if stats.errors > MAXERRORS {
			delete(pm.proxies, prx)
			pm.badProxies[prx] = stats
			pm.rmFromSorted(prx)
			logging.Warn("proxy " + prx.Address + " is removed due to exceeding the error limit (last err \"" + error +"\")")
		}
	}
}

// returns the best available proxy
func (pm *ProxyManager) getBestProxy() *Proxy {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.sortedProxies) == 0 {
		logging.Error("getBestProxy: no available proxies, but requested")
		return nil
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