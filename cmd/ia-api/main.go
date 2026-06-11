// ia-api 是投资助手 HTTP API 入口（H8），供 uni-app 前端调用。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/investment-assistant/investment-assistant/internal/api"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

func main() {
	addr := flag.String("addr", ":8787", "监听地址")
	flag.Parse()

	ac, err := account.ResolveFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	srv, err := api.NewServer(ac, api.Options{Addr: *addr})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := srv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
