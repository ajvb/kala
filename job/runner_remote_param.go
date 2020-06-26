package job

import (
	"encoding/json"
	"io"
	"net/url"
	"reflect"
	"runtime"
	"strings"
)

const (
	mimeFormURLEncoded = "application/x-www-form-urlencoded"
	mimeFormData       = "multipart/form-data"
)

var (
	typeOfPointerToBinaryData = reflect.TypeOf(&BinaryData{})
	typeOfPointerToBinaryFile = reflect.TypeOf(&BinaryFile{})
)

// BinaryData represents binary data from a given source.
type BinaryData struct {
	Filename    string    // filename used in multipart form writer.
	Source      io.Reader // file data source.
	ContentType string    // content type of the data.
}

// BinaryFile represents a file on disk.
type BinaryFile struct {
	Filename    string // filename used in multipart form writer.
	Path        string // path to file. must be readable.
	ContentType string // content type of the file.
}

// Params is the params used to send to API server.
//
// For general uses, just use Params as an ordinary map.
//
// For advanced uses, use MakeParams to create Params from any struct.
type Params map[string]interface{}

// Returns nil if data cannot be used to make a Params instance.
func MakeParams(data interface{}) (params Params) {
	if p, ok := data.(Params); ok {
		return p
	}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}

			params = nil
		}
	}()

	params = makeParams(reflect.ValueOf(data))
	return
}

func makeParams(value reflect.Value) (params Params) {
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	// only map with string keys can be converted to Params
	if value.Kind() == reflect.Map && value.Type().Key().Kind() == reflect.String {
		params = Params{}
		for _, key := range value.MapKeys() {
			params[key.String()] = value.MapIndex(key).Interface()
		}

		return
	}

	if value.Kind() != reflect.Struct {
		return
	}

	params = Params{}
	t := value.Type()
	num := value.NumField()

	for i := 0; i < num; i++ {
		sf := t.Field(i)
		tag := sf.Tag
		name := ""
		omitEmpty := false

		jsonTag := tag.Get("json")

		if jsonTag != "" {
			optTag := jsonTag

			opts := strings.Split(optTag, ",")

			if opts[0] != "" {
				name = opts[0]
			}

			for _, opt := range opts[1:] {
				if opt == "omitempty" {
					omitEmpty = true
				}
			}
		}

		field := value.Field(i)

		if omitEmpty && isEmptyValue(field) {
			continue
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		// If name is not set in field tag, use field name directly.
		if name == "" {
			name = sf.Name
		}

		switch field.Kind() {
		case reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Invalid:
			// these types won't be marshalled in json.
			params = nil
			return

		case reflect.Struct:
			params[name] = makeParams(field)

		default:
			params[name] = field.Interface()
		}
	}

	return
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return false
}

// Encode encodes params to query string.
// If map value is not a string, Encode uses json.Marshal() to convert value to string.
//
// Encode may panic if Params contains values that cannot be marshalled to json string.
func (params Params) Encode(writer io.Writer) (mime string, err error) {
	if params == nil || len(params) == 0 {
		mime = mimeFormURLEncoded
		return
	}

	for _, v := range params {
		typ := reflect.TypeOf(v)

		if typ == typeOfPointerToBinaryData || typ == typeOfPointerToBinaryFile {
			//hasBinary
			break
		}
	}

	return params.encodeFormURLEncoded(writer)
}

func (params Params) encodeFormURLEncoded(writer io.Writer) (mime string, err error) {
	var jsonStr []byte
	written := false

	for k, v := range params {
		if v == nil {
			continue
		}

		if written {
			io.WriteString(writer, "&")
		}

		io.WriteString(writer, url.QueryEscape(k))
		io.WriteString(writer, "=")

		if reflect.TypeOf(v).Kind() == reflect.String {
			io.WriteString(writer, url.QueryEscape(reflect.ValueOf(v).String()))
		} else {
			jsonStr, err = json.Marshal(v)
			if err != nil {
				return
			}
			io.WriteString(writer, url.QueryEscape(string(jsonStr)))
		}

		written = true
	}

	mime = mimeFormURLEncoded
	return
}
