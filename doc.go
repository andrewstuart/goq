// Package goq was built to allow users to declaratively unmarshal HTML into go
// structs using struct tags composed of css selectors.
//
// I've made a best effort to behave very similarly to JSON and XML decoding as
// well as exposing as much information as possible in the event of an error to
// help you debug your Unmarshaling issues.
//
// When creating struct types to be unmarshaled into, the following general
// rules apply:
//
// - Any type that implements the Unmarshaler interface will be passed a slice
// of *html.Node so that manual unmarshaling may be done. This takes the
// highest precedence.
//
// - Any struct fields may be annotated with goquery metadata, which takes the
// form of an element selector followed by arbitrary comma-separated "value
// selectors."
//
// - A value selector may be one of `html`, `text`, or `[someAttrName]`. `html`
// and `text` will result in the methods of the same name being called on the
// `*goquery.Selection` to obtain the value. `[someAttrName]` will result in
// `*goquery.Selection.Attr("someAttrName")` being called for the value.
//
// - A primitive value type will default to the text value of the resulting
// nodes if no value selector is given.
//
// - At least one value selector is required for maps, to determine the map key.
// The key type must follow both the rules applicable to go map indexing, as
// well as these unmarshaling rules. The value of each key will be unmarshaled
// in the same way the element value is unmarshaled.
//
// - For maps, keys will be retreived from the *same level* of the DOM. The key
// selector may be arbitrarily nested, though. The first level of children
// with any number of matching elements will be used, though.
//
// - For maps, any values *must* be nested *below* the level of the key
// selector. Parents or siblings of the element matched by the key selector will
// not be considered.
//
// - Once used, a "value selector" will be shifted off of the comma-separated
// list. This allows you to nest arbitrary levels of value selectors. For
// example, the type `[]map[string][]string` would require one selector for the
// map key, and take an optional second selector for the values of the string
// slice.
//
// - Any struct type encountered in nested types (e.g. map[string]SomeStruct)
// will override any remaining "value selectors" that had not been used. For
// example, given:
//   struct S {
//     F string `goquery:",[bang]"`
//   }
//
//   struct {
//     T map[string]S `goquery:"#someId,[foo],[bar],[baz]"`
//   }
// `[foo]` will be used to determine the string map key,but `[bar]` and `[baz]`
// will be ignored, with the `[bang]` tag present S struct type taking
// precedence.
package goq
