package main

import (
	"github.com/carlosperez-dev/playita_cli/cmd"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cmd.Execute()
}
