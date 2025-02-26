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
package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/etidart/proxyflow/internal/connector"
	"github.com/etidart/proxyflow/internal/logging"
	"github.com/etidart/proxyflow/internal/proxy"
)

const (
	REQUESTGRANTED byte = 0x00
	GENERALFAILURE byte = 0x01
	NOTALLOWED byte = 0x02
	NETUNREACH byte = 0x03
	HOSTUNREACH byte = 0x04
	CONNREFUSED byte = 0x05
	TTLEXPIRED byte = 0x06
	PROTOERR byte = 0x07
	ADDRTYPEERR byte = 0x08
)

const (
	MAXRETRIES = 2
)

func ListenAndServe(listenon string, rqc chan<- chan proxy.Message) {
	listener, err := net.Listen("tcp", listenon)
	if err != nil {
		logging.Fatal("listener: " + err.Error())
	}
	defer listener.Close()
	logging.Info("Started listening on " + listenon)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logging.Warn("error while accepting connection: " + err.Error())
			continue
		}
		go handleConn(conn, rqc)
	}
}

func handleConn(conn net.Conn, rqc chan<- chan proxy.Message) {
	defer conn.Close()
	buff := make([]byte, 4096)
	_, err := conn.Read(buff)
	if err != nil {
		return
	}
	if buff[0] != 0x05 {
		return
	}
	_, err = conn.Write([]byte{0x05,0x00})
	if err != nil {
		return
	}
	_, err = conn.Read(buff)
	if err != nil {
		return
	}
	// parsing
	if buff[0] != 0x05 || buff[1] != 0x01 || buff[2] != 0x00 {
		endHandshake(PROTOERR, conn)
		return
	}

	var rqhost connector.ConnectWho
	var hosttodisplay string
	switch buff[3] {
	case 0x01: // ipv4
		rqhost.IP = fmt.Sprintf("%d.%d.%d.%d", buff[4], buff[5], buff[6], buff[7])
		rqhost.Port = binary.BigEndian.Uint16(buff[8:10])
		hosttodisplay = fmt.Sprintf("%s:%d", rqhost.IP, rqhost.Port)
	case 0x03: // hostname
		size := buff[4]
		host := string(buff[5:5+size])
		rqhost.Port = binary.BigEndian.Uint16(buff[5+size:7+size])
		hosttodisplay = fmt.Sprintf("%s:%d", host, rqhost.Port)
		ipAddrs, err := net.LookupIP(host)
		if err != nil {
			endHandshake(HOSTUNREACH, conn)
			logging.Warn("unable to lookup host's ip from request: " + host + "; error:" + err.Error())
			return
		}
		for _, ip := range ipAddrs {
			if ip.To4() != nil {
				rqhost.IP = ip.String()
			}
		}
		if rqhost.IP == "" {
			endHandshake(HOSTUNREACH, conn)
			logging.Warn("unable to lookup host's ip from request: " + host)
			return
		}
	default: // ipv6 and incorrect
		endHandshake(ADDRTYPEERR, conn)
		return
	}
	var pconn net.Conn
	var retrynum uint8 = 0
	for pconn = getpconn(rqc, &rqhost); pconn == nil; {
		retrynum++
		if (retrynum > MAXRETRIES) {
			logging.Error("unable to get proxy for request (" + hosttodisplay + "). dropping request...")
			break
		} else {
			logging.Error("unable to get proxy for request (" + hosttodisplay + "). retrying...")
		}
	}
	if pconn == nil {
		endHandshake(GENERALFAILURE, conn)
		return
	}
	defer pconn.Close()

	if !endHandshake(REQUESTGRANTED, conn) {
		logging.Warn(conn.RemoteAddr().String() + " suddenly closed the connection")
		return
	}
	logging.Info("accepted request from " + conn.RemoteAddr().String() + " (" + hosttodisplay + "; proxy: " + pconn.RemoteAddr().String() + ")")


	var control sync.WaitGroup
	control.Add(2)
	go func() {
		defer control.Done()
		io.Copy(conn, pconn)
	}()
	go func() {
		defer control.Done()
		io.Copy(pconn, conn)
	}()
	control.Wait()
}

func getpconn(rqc chan<- chan proxy.Message, rqhost *connector.ConnectWho) net.Conn {
	c := make(chan proxy.Message)
	rqc <- c
	prx := (<-c).Prx
	if prx == nil {
		return nil
	}
	pconn, perr, ptime := connector.ConnectToPrx(prx, *rqhost)
	if perr != "" {
		c <- proxy.Message{
			Prx: prx,
			Err: perr,
			Dur: 0,
		}
		return nil
	}
	c <- proxy.Message{
		Prx: prx,
		Err: "",
		Dur: ptime,
	}
	return pconn
}

func endHandshake(status byte, conn net.Conn) bool {
	ans := make([]byte, 0, 10)
	ans = append(ans, 0x05, status, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	_, err := conn.Write(ans)
	return err == nil
}