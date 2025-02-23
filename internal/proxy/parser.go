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
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/etidart/proxyflow/internal/logging"
)

func parseLine(line string) (string, Protocol, bool) {
	if strings.HasPrefix(line, "http://") {
		return strings.TrimPrefix(line, "http://"), HTTP, false
	} else if strings.HasPrefix(line, "https://") {
		return strings.TrimPrefix(line, "https://"), HTTPS, false
	} else if strings.HasPrefix(line, "socks4://") {
		return strings.TrimPrefix(line, "socks4://"), SOCKS4, false
	} else if strings.HasPrefix(line, "socks5://") {
		return strings.TrimPrefix(line, "socks5://"), SOCKS5, false
	}
	return "", 0, true
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

func (pm *ProxyManager) ParseFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		addr, prot, err := parseLine(line)
		if err {
			logging.Warn(fmt.Sprintf("unknown proto in %s, line %d", filename, lineNumber))
			continue
		}

		if !isValidAddress(addr) {
			logging.Warn(fmt.Sprintf("invalid addr format in %s, line %d", filename, lineNumber))
			continue
		}

		pm.AddProxy(addr, prot)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}