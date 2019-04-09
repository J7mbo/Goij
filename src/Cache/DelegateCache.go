package Cache

import (
	"fmt"
	"reflect"
)

type DelegateCache interface {
	Store(string, interface{})
	FindByName(string) interface{}
	FindByValue(reflect.Value) interface{}
	FindByType(reflect.Type) interface{}
}

type delegateCache struct {
	cachedDelegates map[string]interface{}
}

func NewDelegateCache() DelegateCache {
	return &delegateCache{cachedDelegates: make(map[string]interface{})}
}

func (r *delegateCache) Store(objName string, factory interface{}) {
	if reflect.TypeOf(factory).Kind() != reflect.Func {
		panic("You can only delegate a function as a factory method for type: " + objName)
	}

	r.cachedDelegates[objName] = factory
}

func (r *delegateCache) FindByName(name string) interface{} {
	if theObject, exists := r.cachedDelegates[name]; exists {
		return toStructPointer(theObject)
	}

	return nil
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *delegateCache) FindByValue(objType reflect.Value) interface{} {
	typeName := reflect.TypeOf(objType.Interface()).Elem().Name()
	typePkg := reflect.TypeOf(objType.Interface()).Elem().PkgPath()

	return r.FindByName(fmt.Sprintf("%s.%s", typePkg, typeName))
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *delegateCache) FindByType(objType reflect.Type) interface{} {
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	return r.FindByName(fmt.Sprintf("%s.%s", objType.PkgPath(), objType.Name()))
}
