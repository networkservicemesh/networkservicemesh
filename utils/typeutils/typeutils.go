package typeutils

import "reflect"

// GetTypeName return the type of the underlying struct for an interface, with a * if a ptr
func GetTypeName(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()

}
