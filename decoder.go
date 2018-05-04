package goq

import (
	"io"

	"github.com/PuerkitoBio/goquery"
)

// Decoder implements the same API you will see in encoding/xml and
// encoding/json except that we do not currently support proper streaming
// decoding as it is not supported by goquery upstream.
type Decoder struct {
	err error
	doc *goquery.Document
}

// NewDecoder returns a new decoder given an io.Reader
func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{}
	d.doc, d.err = goquery.NewDocumentFromReader(r)
	return d
}

// Decode will unmarshal the contents of the decoder when given an instance of
// an annotated type as its argument. It will return any errors encountered
// during either parsing the document or unmarshaling into the given object.
func (d *Decoder) Decode(dest interface{}) error {
	if d.err != nil {
		return d.err
	}
	if d.doc == nil {
		return &CannotUnmarshalError{
			Reason: "resulting document was nil",
		}
	}

	return UnmarshalSelection(d.doc.Selection, dest)
}
