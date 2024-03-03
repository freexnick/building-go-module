package main

import "toolkit"

func main() {
	var tools toolkit.Tools
	tools.CreateDirIfNotExist("./test-dir")
}
