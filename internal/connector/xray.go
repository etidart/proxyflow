//go:build xray

/*
 * Copyright (C) 2026 Arseniy Astankov
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
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

func xrayHandshake(address string, connTo ConnectWho) (net.Conn, string) {
	xray_port, username, _ := strings.Cut(address, ":")
	conn, err := net.Dial("tcp4", "127.0.0.1:"+fmt.Sprint(xray_port))
	if err != nil {
		panic(err)
	}

	//stage 1
	s1rq := []byte{0x05, 0x01, 0x02}
	_, err = conn.Write(s1rq)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage1s: " + err.Error()
	}
	buff := make([]byte, 16384)
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage1r: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x05, 0x02}) {
		conn.Close()
		return nil, "crit: xray stage1r: auth is not accepted"
	}
	//stage 1_auth
	s1arq := make([]byte, 0, 4+len(username))
	s1arq = append(s1arq, 0x01, byte(len(username)))
	s1arq = append(s1arq, []byte(username)...)
	s1arq = append(s1arq, 0x01)
	s1arq = append(s1arq, []byte("1")...)
	_, err = conn.Write(s1arq)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage1as: " + err.Error()
	}
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage1ar: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x01, 0x00}) {
		conn.Close()
		return nil, "crit: xray stage1r: auth is not accepted"
	}
	//stage 2
	s2rq := make([]byte, 0, 10)
	s2rq = append(s2rq, 0x05, 0x01, 0x00, 0x01)
	rip := net.ParseIP(connTo.IP).To4()
	s2rq = append(s2rq, rip...)
	s2rq = binary.BigEndian.AppendUint16(s2rq, connTo.Port)
	_, err = conn.Write(s2rq)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage2s: " + err.Error()
	}
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: xray stage2r: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x05, 0x00, 0x00}) {
		conn.Close()
		return nil, "xray stage2r: answer is not 00h (granted)"
	}
	// done -----------------
	return conn, ""
}
