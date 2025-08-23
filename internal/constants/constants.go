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
package constants

import "time"

// checker/
const (
	CHKHOST = "www.gstatic.com" // host: while checking, what host should be reached (port is always 443 and proto is always https)
	CHKURL = "/generate_204" // url: ..., what url should be reached
	CHKUSERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36" // useragent: ..., with what useragent should be request with
	CHKRESPCODE = 204 // response code: ..., what response code should be considered good
	CHKTO = time.Duration(1) * time.Second // timeout: ..., how much time is acceptable for TLS handshake with host, sending request and getting response. NOT COVERING CONNECTION TO PROXY (AND CONNECTION FROM PROXY TO HOST)
	CHKTOBTWNCHKS = time.Duration(2) * time.Second // timeout between checks: how much time should pass after each check ended before a new check started in each goroutine separately. it is only needed to not overload network by too many requestes.
)
// /checker

// connector/
const (
	CONCONNHSTO = time.Duration(1) * time.Second // connect+handshake timeout: how much time is acceptable for connecting to a proxy and, separately: possible TLS handshaking, sending requested address, waiting till proxy answers (which implies time to connect to a requested host from proxy).
	CONUSERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36" // useragent: what useragent to send when connecting to http/https proxies.
)
// /connector

// proxy/
const (
	PRXMAXERRS = 2 // max errors: how many errors can proxy get before sending to badProxies
	PRXCHKCD = time.Duration(1) * time.Second // check cooldown: how much time should pass before each proxy separately can be checked again. it is only needed to not overload single proxy which can be loaded by user requests meanwhile
	PRXDEFHSAVG = CONCONNHSTO + time.Duration(1) * time.Second // default handshake average: when new proxy is added, what should be its handshakeAvg value. must be greater than CONCONNHSTO for optimized working
	PRXMINHSAVGDIFF = time.Duration(500) * time.Millisecond // min handshakeAvg difference: on what minimal difference handshakeAvg should be updated
)
// /proxy

// server/
const (
	SRVMAXRETRIES = 3 // max retries: how many times can server retry to get a working proxy
	SRVRETRYCD = time.Duration(500) * time.Millisecond // retry cooldown: how much time should pass before retrying again to get a working proxy
)
// /server