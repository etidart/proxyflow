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
package connector

import (
	"net"
	"time"

	"github.com/etidart/proxyflow/internal/proxy"
)

type ConnectWho struct {
	IP string
	Port uint16
}

func ConnectToPrx(prx *proxy.Proxy, connTo ConnectWho) (net.Conn, string, time.Duration) {
	currTime := time.Now()
	// time is measuring -------------
	operationsTO, _ := time.ParseDuration("1s") // timeout for handshaking (and connecting)
	connection, err := net.DialTimeout("tcp4", prx.Address, operationsTO)
	if err != nil {
		return nil, "while connecting: " + err.Error(), 0
	}
	connection.SetDeadline(time.Now().Add(operationsTO))

	// transfering all the work
	var rconn net.Conn
	var rerr string
	switch prx.Proto {
	case proxy.HTTP:
		rconn, rerr = httpHandshake(connection, connTo)
	case proxy.HTTPS:
		rconn, rerr = httpsHandshake(connection, connTo)
	case proxy.SOCKS4:
		rconn, rerr = s4Handshake(connection, connTo)
	case proxy.SOCKS5:
		rconn, rerr = s5Handshake(connection, connTo)
	}
	// -------------------------------
	if rerr == "" {
		hsMeasure := time.Since(currTime)
		return rconn, "", hsMeasure
	}
	return nil, rerr, 0
}