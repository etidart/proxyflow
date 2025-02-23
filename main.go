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
package main

import (
	"flag"

	"github.com/etidart/proxyflow/internal/checker"
	"github.com/etidart/proxyflow/internal/logging"
	"github.com/etidart/proxyflow/internal/proxy"
	"github.com/etidart/proxyflow/internal/server"
)

func main() {
	logging.Init()
	pfile := flag.String("pfile", "", "path to file containing proxies")
	checkingn := flag.Int("chkth", 10, "number of threads in checking pool")
	listenon := flag.String("listen", "127.0.0.1:1080", "address to listen on")
	flag.Parse()
	if *pfile == "" {
		logging.Fatal("pfile arg is empty")
	}

	pm := proxy.NewProxyManager()
	err := pm.ParseFile(*pfile)
	if err != nil {
		logging.Fatal("got err while parsing " + *pfile + " :" + err.Error())
	}
	checker.StartChecking(*checkingn, pm)

	proxiesChannel := make(chan chan proxy.Message)
	go pm.ServeProxies(proxiesChannel)

	server.ListenAndServe(*listenon, proxiesChannel)
}