//go:build xray

/*
 * Copyright (C) 2025-2026 Arseniy Astankov
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
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/etidart/proxyflow/internal/logging"
	"github.com/xtls/libxray/share"
	"github.com/xtls/libxray/xray"
)

func parseLine(line string) (string, Protocol, bool) {
	if after, ok := strings.CutPrefix(line, "http://"); ok {
		return after, HTTP, false
	} else if after, ok := strings.CutPrefix(line, "https://"); ok {
		return after, HTTPS, false
	} else if after, ok := strings.CutPrefix(line, "socks4://"); ok {
		return after, SOCKS4, false
	} else if after, ok := strings.CutPrefix(line, "socks5://"); ok {
		return after, SOCKS5, false
	} else if after, ok := strings.CutPrefix(line, "socks://"); ok {
		return after, SOCKS5, false
	} else {
		conf, err := share.ConvertShareLinksToXrayJson(line)
		if err != nil {
			return "", 0, true
		}
		str_conf, _ := json.Marshal(conf.OutboundConfigs[0])
		return string(str_conf), XRAY, false
	}
}

func isValidAddress(addr string) bool {
	// split the address into host and port
	hostPort := strings.Split(addr, ":")
	if len(hostPort) != 2 {
		return false
	}

	host := hostPort[0]
	portStr := hostPort[1]

	// validate the IP address
	if net.ParseIP(host) == nil {
		return false
	}

	// validate the port number
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return false
	}

	return true
}

func getFreePort() (int, error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func launchXray(outbound_cfgs []string) int {
	cfg := "{\"inbounds\":[{\"tag\":\"socks-inbound\",\"port\":"
	free_port, err := getFreePort()
	if err != nil {
		panic(err)
	}
	cfg += fmt.Sprint(free_port)
	cfg += ",\"listen\":\"127.0.0.1\",\"protocol\":\"socks\",\"settings\":{\"auth\":\"password\",\"accounts\":["
	for i := range len(outbound_cfgs) {
		cfg += fmt.Sprintf("{\"user\":\"ob_%d\",\"pass\":\"1\"}", i)
		if i != len(outbound_cfgs)-1 {
			cfg += ","
		}
	}
	cfg += "],\"udp\":true}}],\"outbounds\":["
	for i, ob := range outbound_cfgs {
		cfg += strings.Replace(ob, "\"tag\":\"\"", fmt.Sprintf("\"tag\":\"ob_%d\"", i), 1)
		if i != len(outbound_cfgs)-1 {
			cfg += ","
		}
	}
	cfg += "],\"routing\":{\"domainStrategy\":\"AsIs\",\"rules\":["
	for i := range len(outbound_cfgs) {
		c_ob := fmt.Sprintf("ob_%d", i)
		cfg += "{\"type\":\"field\",\"user\":[\"" + c_ob + "\"],\"outboundTag\":\"" + c_ob + "\"}"
		if i != len(outbound_cfgs)-1 {
			cfg += ","
		}
	}
	cfg += "]}}"

	err = xray.RunXrayFromJSON("", cfg)
	if err != nil {
		panic(err)
	}

	return free_port
}

func (pm *ProxyManager) ParseFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	xray_ob_cfgs := []string{}
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		addr, prot, err := parseLine(line)
		if err {
			logging.Warn(fmt.Sprintf("unknown proto in %s, line %d", filename, lineNumber))
			continue
		}

		if prot != XRAY {
			if !isValidAddress(addr) {
				logging.Warn(fmt.Sprintf("invalid addr format in %s, line %d", filename, lineNumber))
				continue
			}

			pm.AddProxy(addr, prot)
		} else {
			xray_ob_cfgs = append(xray_ob_cfgs, addr)
		}
	}

	if len(xray_ob_cfgs) != 0 {
		port := launchXray(xray_ob_cfgs)
		for i := range len(xray_ob_cfgs) {
			pm.AddProxy(fmt.Sprintf("%d:ob_%d", port, i), XRAY)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
