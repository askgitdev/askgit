package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsImage struct{}

func (f *EnryIsImage) Args() int           { return 1 }
func (f *EnryIsImage) Deterministic() bool { return true }
func (f *EnryIsImage) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsImage(value[0].Text()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
