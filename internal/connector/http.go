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
	"bytes"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/etidart/proxyflow/internal/constants"
)

func httpHandshake(conn net.Conn, connTo ConnectWho) (net.Conn, string) {
	tosend := fmt.Sprintf("CONNECT %[1]s:%[2]d HTTP/1.1\r\nHost: %[1]s:%[2]d\r\nUser-Agent: %[3]s\r\nProxy-Connection: Keep-Alive\r\n\r\n",
						  connTo.IP, connTo.Port, constants.CONUSERAGENT)
	_, err := conn.Write([]byte(tosend))
	if err != nil {
		conn.Close()
		return nil, "crit: http stage1s: " + err.Error()
	}
	buff := make([]byte, 16384)
	n, err := conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: http stage1r: " + err.Error()
	}
	shouldbe := []byte("HTTP/1.1 200")
	if n < len(shouldbe) || !bytes.HasPrefix(buff, shouldbe) {
		conn.Close()
		return nil, "http stage1r: answer is not 200 OK"
	}
	// done -----------------
	return conn, ""
}

func httpsHandshake(conn net.Conn, connTo ConnectWho) (net.Conn, string) {
	tlsConn := tls.Client(conn, getTLSConfig())
	err := tlsConn.Handshake()
	if err != nil {
		conn.Close()
		return nil, "crit: https tls handshake: " + err.Error()
	}
	return httpHandshake(tlsConn, connTo)
}