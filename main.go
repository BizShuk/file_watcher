package main

import (
	"github.com/bizshuk/file_watcher/cmd"
	"github.com/bizshuk/file_watcher/config"
)

func main() {
	config.Default()
	cmd.Execute()
}
