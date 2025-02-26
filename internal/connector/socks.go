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
	"encoding/binary"
	"net"
	"time"
)

func s4Handshake(conn net.Conn, connTo ConnectWho) (net.Conn, string) {
	rip := net.ParseIP(connTo.IP).To4()
	request := make([]byte, 0, 9)
	request = append(request, 0x04, 0x01)
	request = binary.BigEndian.AppendUint16(request, connTo.Port)
	request = append(request, rip...)
	request = append(request, 0x00)

	_, err := conn.Write(request)
	if err != nil {
		conn.Close()
		return nil, "crit: s4 stage1s: " + err.Error()
	}
	buff := make([]byte, BUFFSIZE)
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: s4 stage1r: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x00, 0x5a}) {
		conn.Close()
		return nil, "s4 stage1r: answer is not 5ah (granted)"
	}
	// done -----------------
	conn.SetDeadline(time.Time{}) // no more deadlines
	return conn, ""
}

func s5Handshake(conn net.Conn, connTo ConnectWho) (net.Conn, string) {
	//stage 1
	s1rq := []byte{0x05, 0x01, 0x00}
	_, err := conn.Write(s1rq)
	if err != nil {
		conn.Close()
		return nil, "crit: s5 stage1s: " + err.Error()
	}
	buff := make([]byte, BUFFSIZE)
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: s5 stage1r: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x05, 0x00}) {
		conn.Close()
		return nil, "crit: s5 stage1r: auth is not accepted"
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
		return nil, "crit: s5 stage2s: " + err.Error()
	}
	_, err = conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, "crit: s5 stage2r: " + err.Error()
	}
	if !bytes.HasPrefix(buff, []byte{0x05, 0x00, 0x00}) {
		conn.Close()
		return nil, "s5 stage2r: answer is not 00h (granted)"
	}
	// done -----------------
	conn.SetDeadline(time.Time{}) // no more deadlines
	return conn, ""
}