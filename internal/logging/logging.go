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
package logging

import (
	"log"
	"os"
)

var logger *log.Logger

func Init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func Info(msg string) {
	logger.Println("INFO", msg)
}
func Warn(msg string) {
	logger.Println("WARN", msg)
}
func Error(msg string) {
	logger.Println("ERROR", msg)
}
func Fatal(msg string) {
	logger.Fatalln("FATAL", msg)
}