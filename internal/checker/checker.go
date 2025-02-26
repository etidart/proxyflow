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
package checker

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/etidart/proxyflow/internal/connector"
	"github.com/etidart/proxyflow/internal/logging"
	"github.com/etidart/proxyflow/internal/proxy"
)

var (
	once sync.Once
	connwho *connector.ConnectWho
)
const (
	HOSTTOCHECK = "www.gstatic.com"
	URLTOCHECK = "/generate_204"
	WANTEDCODE = 204
	USERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36" // the most popular one
)

func getconnto() *connector.ConnectWho {
	once.Do(func() {
		connwho = &connector.ConnectWho{
			IP: "",
			Port: 443,
		}
		ipAddresses, err := net.LookupIP(HOSTTOCHECK)
    	if err != nil {
			logging.Fatal("IP of " + HOSTTOCHECK + " wasn't resolved: " + err.Error())
    	}
		for _, ip := range ipAddresses {
			if ip.To4() != nil {
				connwho.IP = ip.String()
				break
			}
		}
		if connwho.IP == "" {
			logging.Fatal("IP of " + HOSTTOCHECK + " wasn't resolved")
		}
	})
	return connwho
}

func check(prx *proxy.Proxy) (string, time.Duration) {
	conn, err, dur := connector.ConnectToPrx(prx, *getconnto())
	if err != "" {
		return "crit (checking phase): " + err, dur
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Duration(1) * time.Second)) // 1sec

	tlsConn := tls.Client(conn, &tls.Config{ServerName: HOSTTOCHECK})
	erro := tlsConn.Handshake()
	if erro != nil {
		return "crit (checking phase): handshaking with remote: " + erro.Error(), dur
	}

	rq := []byte(fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nAccept: */*\r\n\r\n", URLTOCHECK, HOSTTOCHECK, USERAGENT))
	shouldbe := []byte(fmt.Sprintf("HTTP/1.1 %d", WANTEDCODE))
	buff := make([]byte, 4096)
	
	_, erro = tlsConn.Write(rq)
	if erro != nil {
		return "crit (checking phase): sending request to remote: " + erro.Error(), dur
	}

	n, erro := tlsConn.Read(buff)
	if erro != nil {
		return "crit (checking phase): getting answer from remote: " + erro.Error(), dur
	}

	if n < len(shouldbe) || !bytes.HasPrefix(buff, shouldbe) {
		return "crit (checking phase): didn't get satisfying answer", dur
	}
	
	return "", dur
}

func checking(rq chan<- chan proxy.Message) {
	for {
		c := make(chan proxy.Message)
		rq <- c
		prx := (<-c).Prx
		err, dur := check(prx)
		c <- proxy.Message{
			Prx: prx,
			Err: err,
			Dur: dur,
		}
		time.Sleep(time.Duration(rand.Intn(3000-1000+1) + 1000) * time.Millisecond)
	}
}

func StartChecking(nth int, pm *proxy.ProxyManager) {
	c := make(chan chan proxy.Message)
	go pm.ServeChecker(c)
	for i := 0; i < nth; i++ {
		go checking(c)
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}