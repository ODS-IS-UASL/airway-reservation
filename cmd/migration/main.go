package main

import (
	"fmt"
	"os"

	"uasl-reservation/cmd/migration/db"
)

func main() {
	if err := db.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
