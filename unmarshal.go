package goq

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"

	"golang.org/x/net/html"
)

// Unmarshaler allows for custom implementations of unmarshaling logic
type Unmarshaler interface {
	UnmarshalHTML([]*html.Node) error
}

// NodeSelector is a quick utility function to get a goquery.Selection from a
// slice of *html.Node. Useful for performing unmarshaling, since the decision
// was made to use []*html.Node for maximum flexibility.
func NodeSelector(nodes []*html.Node) *goquery.Selection {
	sel := &goquery.Selection{}
	return sel.AddNodes(nodes...)
}

type valFunc func(*goquery.Selection) string

type goqueryTag string

const (
	prePfx    = '!'
	tagName   = "goquery"
	ignoreTag = "!ignore"
)

func (tag goqueryTag) preprocess(s *goquery.Selection) *goquery.Selection {
	arr := strings.Split(string(tag), ",")
	var offset int
	for len(arr)-1 > offset && arr[offset][0] == prePfx {
		meth := arr[offset][1:]
		v := reflect.ValueOf(s).MethodByName(meth)
		if !v.IsValid() {
			return s
		}

		result := v.Call(nil)

		if sel, ok := result[0].Interface().(*goquery.Selection); ok {
			s = sel
		}
		offset++
	}
	return s
}

func (tag goqueryTag) selector(which int) string {
	arr := strings.Split(string(tag), ",")
	if which > len(arr)-1 {
		return ""
	}
	var offset int
	for len(arr) > offset && arr[offset][0] == prePfx {
		offset++
	}
	return arr[which+offset]
}

var (
	textVal valFunc = func(s *goquery.Selection) string {
		return strings.TrimSpace(s.Text())
	}
	htmlVal = func(s *goquery.Selection) string {
		str, _ := s.Html()
		return strings.TrimSpace(str)
	}

	vfMut   = sync.Mutex{}
	vfCache = map[goqueryTag]valFunc{}
)

func attrFunc(attr string) valFunc {
	return func(s *goquery.Selection) string {
		str, _ := s.Attr(attr)
		return str
	}
}

func (tag goqueryTag) valFunc() valFunc {
	vfMut.Lock()
	defer vfMut.Unlock()

	if fn := vfCache[tag]; fn != nil {
		return fn
	}

	srcArr := strings.Split(string(tag), ",")
	if len(srcArr) < 2 {
		vfCache[tag] = textVal
		return textVal
	}

	src := srcArr[1]

	var f valFunc
	switch {
	case src[0] == '[':
		// [someattr] will return value of .Attr("someattr")
		attr := src[1 : len(src)-1]
		f = attrFunc(attr)
	case src == "html":
		f = htmlVal
	case src == "text":
		f = textVal
	default:
		f = textVal
	}

	vfCache[tag] = f
	return f
}

// popVal should allow us to handle arbitrarily nested maps as well as the
// cleanly handling the possiblity of map[literal]literal by just delegating
// back to `unmarshalByType`.
func (tag goqueryTag) popVal() goqueryTag {
	arr := strings.Split(string(tag), ",")
	if len(arr) < 2 {
		return tag
	}
	newA := []string{arr[0]}
	newA = append(newA, arr[2:]...)

	return goqueryTag(strings.Join(newA, ","))
}

// Unmarshal takes a byte slice and a destination pointer to any
// interface{}, and unmarshals the document into the destination based on the
// rules above. Any error returned here will likely be of type
// CannotUnmarshalError, though an initial goquery error will pass through
// directly.
func Unmarshal(bs []byte, v interface{}) error {
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(bs))

	if err != nil {
		return err
	}

	return UnmarshalSelection(d.Selection, v)
}

func wrapUnmErr(err error, v reflect.Value) error {
	if err == nil {
		return nil
	}

	return &CannotUnmarshalError{
		V:      v,
		Reason: customUnmarshalError,
		Err:    err,
	}
}

// UnmarshalSelection will unmarshal a goquery.goquery.Selection into an interface
// appropriately annoated with goquery tags.
func UnmarshalSelection(s *goquery.Selection, iface interface{}) error {
	v := reflect.ValueOf(iface)

	// Must come before v.IsNil() else IsNil panics on NonPointer value
	if v.Kind() != reflect.Ptr {
		return &CannotUnmarshalError{V: v, Reason: nonPointer}
	}

	if iface == nil || v.IsNil() {
		return &CannotUnmarshalError{V: v, Reason: nilValue}
	}

	u, v := indirect(v)

	if u != nil {
		return wrapUnmErr(u.UnmarshalHTML(s.Nodes), v)
	}

	return unmarshalByType(s, v, "")
}

func unmarshalByType(s *goquery.Selection, v reflect.Value, tag goqueryTag) error {
	u, v := indirect(v)

	if u != nil {
		return wrapUnmErr(u.UnmarshalHTML(s.Nodes), v)
	}

	// Handle special cases where we can just set the value directly
	switch val := v.Interface().(type) {
	case []*html.Node:
		val = append(val, s.Nodes...)
		v.Set(reflect.ValueOf(val))
		return nil
	}

	t := v.Type()

	switch t.Kind() {
	case reflect.Struct:
		return unmarshalStruct(s, v)
	case reflect.Slice:
		return unmarshalSlice(s, v, tag)
	case reflect.Array:
		return unmarshalArray(s, v, tag)
	case reflect.Map:
		return unmarshalMap(s, v, tag)
	default:
		vf := tag.valFunc()
		str := vf(s)
		err := unmarshalLiteral(str, v)
		if err != nil {
			return &CannotUnmarshalError{
				V:      v,
				Reason: typeConversionError,
				Err:    err,
				Val:    str,
			}
		}
		return nil
	}
}

func unmarshalLiteral(s string, v reflect.Value) error {
	t := v.Type()

	switch t.Kind() {
	case reflect.Interface:
		if t.NumMethod() == 0 {
			// For empty interfaces, just set to a string
			nv := reflect.New(reflect.TypeOf(s)).Elem()
			nv.Set(reflect.ValueOf(s))
			v.Set(nv)
		}
	case reflect.Bool:
		i, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(i)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Float32, reflect.Float64:
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(i)
	case reflect.String:
		v.SetString(s)
	}
	return nil
}

func unmarshalStruct(s *goquery.Selection, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		tag := goqueryTag(t.Field(i).Tag.Get(tagName))

		if tag == ignoreTag {
			continue
		}

		// If tag is empty and the object doesn't implement Unmarshaler, skip
		if tag == "" {
			if u, _ := indirect(v.Field(i)); u == nil {
				continue
			}
		}

		sel := tag.preprocess(s)
		if tag != "" {
			selStr := tag.selector(0)
			sel = sel.Find(selStr)
		}

		err := unmarshalByType(sel, v.Field(i), tag)
		if err != nil {
			return &CannotUnmarshalError{
				Reason:   typeConversionError,
				Err:      err,
				V:        v,
				FldOrIdx: t.Field(i).Name,
			}
		}
	}
	return nil
}

func unmarshalArray(s *goquery.Selection, v reflect.Value, tag goqueryTag) error {
	if v.Type().Len() != len(s.Nodes) {
		return &CannotUnmarshalError{
			Reason: arrayLengthMismatch,
			V:      v,
		}
	}

	for i := 0; i < v.Type().Len(); i++ {
		err := unmarshalByType(s.Eq(i), v.Index(i), tag)
		if err != nil {
			return &CannotUnmarshalError{
				Reason:   typeConversionError,
				Err:      err,
				V:        v,
				FldOrIdx: i,
			}
		}
	}

	return nil
}

func unmarshalSlice(s *goquery.Selection, v reflect.Value, tag goqueryTag) error {
	slice := v
	eleT := v.Type().Elem()

	for i := 0; i < s.Length(); i++ {
		newV := reflect.New(TypeDeref(eleT))

		err := unmarshalByType(s.Eq(i), newV, tag)

		if err != nil {
			return &CannotUnmarshalError{
				Reason:   typeConversionError,
				Err:      err,
				V:        v,
				FldOrIdx: i,
			}
		}

		if eleT.Kind() != reflect.Ptr {
			newV = newV.Elem()
		}

		v = reflect.Append(v, newV)
	}

	slice.Set(v)
	return nil
}

func childrenUntilMatch(s *goquery.Selection, sel string) *goquery.Selection {
	orig := s
	s = s.Children()
	for s.Length() != 0 && s.Filter(sel).Length() == 0 {
		s = s.Children()
	}
	if s.Length() == 0 {
		return orig
	}
	return s.Filter(sel)
}

func unmarshalMap(s *goquery.Selection, v reflect.Value, tag goqueryTag) error {
	// Make new map here because indirect for some Reason doesn't help us out
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	keyT, eleT := v.Type().Key(), v.Type().Elem()

	if tag.selector(1) == "" {
		// We need minimum one value selector to determine the map key
		return &CannotUnmarshalError{
			Reason: missingValueSelector,
			V:      v,
		}
	}

	valTag := tag

	// Find children at the same level that match the given selector
	s = childrenUntilMatch(s, tag.selector(1))
	// Then augment the selector we will pass down to the next unmarshal step
	valTag = valTag.popVal()

	var err error
	s.EachWithBreak(func(_ int, subS *goquery.Selection) bool {
		newK, newV := reflect.New(TypeDeref(keyT)), reflect.New(TypeDeref(eleT))

		err = unmarshalByType(subS, newK, tag)
		if err != nil {
			err = &CannotUnmarshalError{
				Reason:   mapKeyUnmarshalError,
				V:        v,
				Err:      err,
				FldOrIdx: newK.Interface(),
				Val:      valTag.valFunc()(subS),
			}
			return false
		}

		err = unmarshalByType(subS, newV, valTag)
		if err != nil {
			return false
		}

		if eleT.Kind() != reflect.Ptr {
			newV = newV.Elem()
		}
		if keyT.Kind() != reflect.Ptr {
			newK = newK.Elem()
		}

		v.SetMapIndex(newK, newV)

		return true
	})

	if err != nil {
		return &CannotUnmarshalError{
			Reason: typeConversionError,
			Err:    err,
			V:      v,
		}
	}

	return nil
}
