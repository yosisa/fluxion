package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompile(t *testing.T) {
	re, _ := compileTag("foo")
	assert.Equal(t, `^foo$`, re.String())

	re, _ = compileTag("foo.bar")
	assert.Equal(t, `^foo\.bar$`, re.String())

	re, _ = compileTag("foo.*.bar")
	assert.Equal(t, `^foo\..*\.bar$`, re.String())

	re, _ = compileTag("foo.**")
	assert.Equal(t, `^foo(\..+|)$`, re.String())

	re, _ = compileTag("**")
	assert.Equal(t, `^.*$`, re.String())
}
