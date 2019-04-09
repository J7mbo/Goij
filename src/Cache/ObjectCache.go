package Cache

import (
	"fmt"
	"reflect"
)

type ObjectCache interface {
	Store(interface{})
	FindByName(string) interface{}
	FindByValue(reflect.Value) interface{}
	FindByType(reflect.Type) interface{}
}

type objectCache struct {
	cachedObjs map[string]interface{}
}

func NewObjectCache() ObjectCache {
	return &objectCache{cachedObjs: make(map[string]interface{})}
}

/* If pointer passed in, it is dereferenced by getValue() so makes no difference in getting the type name. */
func (r *objectCache) Store(obj interface{}) {
	var typeName string

	typeName = fmt.Sprintf("%s.%s", reflect.TypeOf(r.getValue(obj)).PkgPath(), reflect.TypeOf(r.getValue(obj)).Name())

	r.cachedObjs[typeName] = obj
}

func (r *objectCache) FindByName(name string) interface{} {
	if theObject, exists := r.cachedObjs[name]; exists {
		return toStructPointer(theObject)
	}

	return nil
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *objectCache) FindByValue(objType reflect.Value) interface{} {
	typeName := reflect.TypeOf(objType.Interface()).Elem().Name()
	typePkg := reflect.TypeOf(objType.Interface()).Elem().PkgPath()

	return r.FindByName(fmt.Sprintf("%s.%s", typePkg, typeName))
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *objectCache) FindByType(objType reflect.Type) interface{} {
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	return r.FindByName(fmt.Sprintf("%s.%s", objType.PkgPath(), objType.Name()))
}

/* Given a value, loop through until we get a concrete element out of it. */
func (r *objectCache) getValue(obj interface{}) interface{} {
	return r.getElem(obj).Interface()
}

func (r *objectCache) getElem(obj interface{}) reflect.Value {
	val := reflect.ValueOf(obj)

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val
}

/* Given a struct, create a pointer to it so we can reflect / edit values on it. */
func toStructPointer(obj interface{}) interface{} {
	vp := reflect.New(reflect.TypeOf(obj))
	vp.Elem().Set(reflect.ValueOf(obj))

	return vp.Interface()
}
