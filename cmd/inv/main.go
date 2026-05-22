// inv 是个人投资助手 MVP-1 CLI 入口（04 §五、§二十三 H0）。
package main

import (
	"github.com/investment-assistant/investment-assistant/internal/cli"
)

// version 构建时可通过 -ldflags 覆盖。
var version = "0.1.0-h0"

func main() {
	cli.Version = version
	cli.Execute()
}
