package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsTest struct{}

func (f *EnryIsTest) Args() int           { return 1 }
func (f *EnryIsTest) Deterministic() bool { return true }
func (f *EnryIsTest) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsTest(value[0].Text()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
