package main

/*
#cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
#cgo windows LDFLAGS: -L${SRCDIR}/../../node_modules/.golar-dev -l:node.lib
*/
import "C"

import _ "github.com/auvred/golar/internal/golar"

func main() {}
