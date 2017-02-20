package goq

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecoder(t *testing.T) {
	asrt := assert.New(t)

	var p page

	asrt.NoError(NewDecoder(strings.NewReader(hnPage)).Decode(&p))
	asrt.Len(p.Items, 30)
}
