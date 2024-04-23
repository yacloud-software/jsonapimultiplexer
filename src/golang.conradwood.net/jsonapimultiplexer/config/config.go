package config

import (
	"flag"
)

var (
	use_jsonapidoc = flag.Bool("use_jsonapidoc",false,"if true, render documentation from jsonapidoc instead of built-in renderer")
)

func UseJsonAPIDoc() bool {
	return *use_jsonapidoc
}
