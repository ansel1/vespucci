package maps

import "reflect"

var _ Map = jsonObj{}
var _ Map = reflectMap{}

var _ Slice = jsonArray{}
var _ Slice = reflectSlice{}

type jsonObj map[string]interface{}

func (j jsonObj) Visit(f func(key string, val interface{}) error) error {
	for key, value := range j {
		if e := f(key, value); e != nil {
			return e
		}
	}
	return nil
}

func (j jsonObj) Len() int {
	return len(j)
}

func (j jsonObj) Get(key string) (interface{}, bool) {
	v, p := j[key]
	return v, p
}

type reflectMap reflect.Value

func (m reflectMap) Visit(f func(key string, val interface{}) error) error {
	for _, v := range reflect.Value(m).MapKeys() {
		key := v.String()
		v := reflect.Value(m).MapIndex(reflect.ValueOf(key))
		if e := f(key, v.Interface()); e != nil {
			return e
		}
	}
	return nil
}

func (m reflectMap) Len() int {
	return reflect.Value(m).Len()
}

func (m reflectMap) Get(key string) (interface{}, bool) {
	rv := reflect.Value(m)
	// todo: I'm not sure there is a more efficient way to know whether a key exists in a map via reflection
	for _, kv := range rv.MapKeys() {
		if kv.String() == key {
			return reflect.Value(m).MapIndex(reflect.ValueOf(key)).Interface(), true
		}

	}
	return nil, false
}

type jsonArray []interface{}

func (s reflectSlice) Visit(f func(i int, val interface{}) error) error {
	rv := reflect.Value(s)
	l := rv.Len()
	for i := 0; i < l; i++ {
		if e := f(i, rv.Index(i).Interface()); e != nil {
			return e
		}
	}
	return nil
}

func (s reflectSlice) Len() int {
	return reflect.Value(s).Len()
}

func (s reflectSlice) Get(i int) interface{} {
	return reflect.Value(s).Index(i).Interface()
}

type reflectSlice reflect.Value

func (s jsonArray) Visit(f func(i int, val interface{}) error) error {
	for i := 0; i < len(s); i++ {
		if e := f(i, s[i]); e != nil {
			return e
		}
	}
	return nil
}

func (s jsonArray) Len() int {
	return len(s)
}

func (s jsonArray) Get(i int) interface{} {
	return s[i]
}
