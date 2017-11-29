# goq
[![Build Status](https://travis-ci.org/andrewstuart/goq.svg?branch=master)](https://travis-ci.org/andrewstuart/goq)
[![GoDoc](https://godoc.org/astuart.co/goq?status.svg)](https://godoc.org/astuart.co/goq)
[![Coverage Status](https://coveralls.io/repos/github/andrewstuart/goq/badge.svg?branch=master)](https://coveralls.io/github/andrewstuart/goq?branch=master)
[![Go Report Card](https://goreportcard.com/badge/astuart.co/goq)](https://goreportcard.com/report/astuart.co/goq)

## Example

```go
import (
	"log"
	"net/http"

	"astuart.co/goq"
)

// Structured representation for github file name table
type example struct {
	Title string `goquery:"h1"`
	Files []string `goquery:"table.files tbody tr.js-navigation-item td.content,text"`
}

func main() {
	res, err := http.Get("https://github.com/andrewstuart/goq")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	var ex example
	
	err = goq.NewDecoder(res.Body).Decode(&ex)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(ex.Title, ex.Files)
}
```

## Details

# goq
--
    import "astuart.co/goq"

Package goq was built to allow users to declaratively unmarshal HTML into go
structs using struct tags composed of css selectors.

I've made a best effort to behave very similarly to JSON and XML decoding as
well as exposing as much information as possible in the event of an error to
help you debug your Unmarshaling issues.

When creating struct types to be unmarshaled into, the following general rules
apply:

- Any type that implements the Unmarshaler interface will be passed a slice of
*html.Node so that manual unmarshaling may be done. This takes the highest
precedence.

- Any struct fields may be annotated with goquery metadata, which takes the form
of an element selector followed by arbitrary comma-separated "value selectors."

- A value selector may be one of `html`, `text`, or `[someAttrName]`. `html` and
`text` will result in the methods of the same name being called on the
`*goquery.Selection` to obtain the value. `[someAttrName]` will result in
`*goquery.Selection.Attr("someAttrName")` being called for the value.

- A primitive value type will default to the text value of the resulting nodes
if no value selector is given.

- At least one value selector is required for maps, to determine the map key.
The key type must follow both the rules applicable to go map indexing, as well
as these unmarshaling rules. The value of each key will be unmarshaled in the
same way the element value is unmarshaled.

- For maps, keys will be retreived from the *same level* of the DOM. The key
selector may be arbitrarily nested, though. The first level of children with any
number of matching elements will be used, though.

- For maps, any values *must* be nested *below* the level of the key selector.
Parents or siblings of the element matched by the key selector will not be
considered.

- Once used, a "value selector" will be shifted off of the comma-separated list.
This allows you to nest arbitrary levels of value selectors. For example, the
type `[]map[string][]string` would require one selector for the map key, and
take an optional second selector for the values of the string slice.

- Any struct type encountered in nested types (e.g. map[string]SomeStruct) will
override any remaining "value selectors" that had not been used. For example,
given:

    struct S {
    	F string `goquery:",[bang]"`
    }

    struct {
    	T map[string]S `goquery:"#someId,[foo],[bar],[baz]"`
    }

`[foo]` will be used to determine the string map key,but `[bar]` and `[baz]`
will be ignored, with the `[bang]` tag present S struct type taking precedence.

## Usage

#### func  NodeSelector

```go
func NodeSelector(nodes []*html.Node) *goquery.Selection
```
NodeSelector is a quick utility function to get a goquery.Selection from a slice
of *html.Node. Useful for performing unmarshaling, since the decision was made
to use []*html.Node for maximum flexibility.

#### func  Unmarshal

```go
func Unmarshal(bs []byte, v interface{}) error
```
Unmarshal takes a byte slice and a destination pointer to any interface{}, and
unmarshals the document into the destination based on the rules above. Any error
returned here will likely be of type CannotUnmarshalError, though an initial
goquery error will pass through directly.

#### func  UnmarshalSelection

```go
func UnmarshalSelection(s *goquery.Selection, iface interface{}) error
```
UnmarshalSelection will unmarshal a goquery.goquery.Selection into an interface
appropriately annoated with goquery tags.

#### type CannotUnmarshalError

```go
type CannotUnmarshalError struct {
	Err      error
	Val      string
	FldOrIdx interface{}
}
```

CannotUnmarshalError represents an error returned by the goquery Unmarshaler and
helps consumers in programmatically diagnosing the cause of their error.

#### func (*CannotUnmarshalError) Error

```go
func (e *CannotUnmarshalError) Error() string
```

#### type Decoder

```go
type Decoder struct {
}
```

Decoder implements the same API you will see in encoding/xml and encoding/json
except that we do not currently support proper streaming decoding as it is not
supported by goquery upstream.

#### func  NewDecoder

```go
func NewDecoder(r io.Reader) *Decoder
```
NewDecoder returns a new decoder given an io.Reader

#### func (*Decoder) Decode

```go
func (d *Decoder) Decode(dest interface{}) error
```
Decode will unmarshal the contents of the decoder when given an instance of an
annotated type as its argument. It will return any errors encountered during
either parsing the document or unmarshaling into the given object.

#### type Unmarshaler

```go
type Unmarshaler interface {
	UnmarshalHTML([]*html.Node) error
}
```

Unmarshaler allows for custom implementations of unmarshaling logic

## TODO

- Callable goquery methods with args, via reflection
