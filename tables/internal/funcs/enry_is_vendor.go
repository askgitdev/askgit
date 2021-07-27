package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsVendor struct{}

func (f *EnryIsVendor) Args() int           { return 1 }
func (f *EnryIsVendor) Deterministic() bool { return true }
func (f *EnryIsVendor) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsVendor(value[0].Text()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
