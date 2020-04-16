package provider

import (
	"log"
	"os"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// Dlog is the development logger
var Dlog *log.Logger

// InitDevLog initialized the development log and returns a cleanup function to be passed to "defer".
func InitDevLog() func() {
	f, err := os.OpenFile("dev.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	Dlog = log.New(f, "RAW provider ", log.Ldate|log.Ltime)
	return func() {
		Dlog.Println("Finished")
		f.Close()
	}
}

// DumpCtyPath creates log-friendly represation of a cty.Path value
func DumpCtyPath(in cty.Path) string {
	b := strings.Builder{}
	for i, p := range in {
		switch t := p.(type) {
		case cty.GetAttrStep:
			b.WriteString(t.Name)
		case cty.IndexStep:
			b.WriteString(t.Key.GoString())
		}
		if i < len(in)-1 {
			b.WriteString("/")
		}
	}
	return b.String()
}
