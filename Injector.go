package main

import (
	"fmt"
	"reflect"

	"github.com/j7mbo/Injector/Logger"
	"github.com/j7mbo/Injector/TypeRegistry"
	"github.com/j7mbo/Injector/ouch/again"
)

func main() {
	tr := TypeRegistry.NewFromGeneratedRegistry()
	// tr.AddType(&again.Needthis{I: 22})
	// @todo separate type registry and object registry
	// tr.AddBuildObject(&again.Parent{})
	tr.AddBuildObject(&again.Needthis{I: 40})

	ij := Injector{tr: tr, logger: Logger.NewStdLogger()}

	parent := ij.Make("again.Parent").(*again.Parent)
	parent.Y.I = 99
	fmt.Println(parent.Y.I)
	fmt.Println(ij.Make("again.Parent").(*again.Parent).Y.I) // this should NOT be 99 it should be 40
}

type Injector struct {
	tr     *TypeRegistry.TypeRegistry
	logger *Logger.Logger
}

/* Format: PackageName.StructName. At present this returns the first registered object.*/
func (ij *Injector) Make(name string, option ...int) interface{} {
	ij.log("Injector asked to provision instance of: " + name)

	var objType interface{}

	if len(option) > 0 {
		objType = ij.tr.FindTypeByName(name, option[0])
	} else {
		objType = ij.tr.FindTypeByName(name)
	}

	if objType == nil {
		msg := "No type found in registry for name: " + name + ", did you forget to register it?"

		ij.elog(msg)
		panic(msg)
	}

	/* See if this object is already cached? */
	foundObj := ij.tr.FindBuiltObjectByValue(reflect.ValueOf(objType))

	if foundObj != nil {
		return reflect.ValueOf(foundObj).Elem().Interface()
	}

	var numFields int
	var concreteObj interface{}

	objIsPointer := func(obj interface{}) bool {
		return reflect.ValueOf(objType).Elem().Kind() == reflect.Ptr
	}

	if objIsPointer(objType) {
		ij.log(fmt.Sprintf("Type: %s, (a pointer) found in type registry", name))

		concreteObj = reflect.ValueOf(objType).Elem().Interface()

		numFields = reflect.ValueOf(objType).Elem().Elem().Elem().NumField()
	} else {
		ij.log(fmt.Sprintf("Type: %s, (a struct) found in type registry", name))

		concreteObj = reflect.ValueOf(objType).Interface()

		numFields = reflect.ValueOf(objType).Elem().NumField()
	}

	ij.log(fmt.Sprintf("Type: %s has %d field(s)", name, numFields))

	for i := 0; i < numFields; i++ {
		var field reflect.Value

		if objIsPointer(concreteObj) {
			field = reflect.ValueOf(&concreteObj).Elem().Elem().Elem().Field(i)

			ij.log(fmt.Sprintf("Found field: %s of type: %s on object: %s", reflect.TypeOf(field).Name(), field.Type(), name))
		} else {
			field = reflect.ValueOf(&concreteObj).Elem().Elem().Elem().Field(i)

			ij.log(fmt.Sprintf("Found field: %s of type: %s on object: %s", reflect.TypeOf(field).Name(), field.Type(), name))
		}

		// If field is not a pointer or a struct, like an int etc... SKIP until we have those in a registry as well!!
		if reflect.TypeOf(field).Kind() != reflect.Ptr && reflect.TypeOf(field).Kind() != reflect.Struct {
			ij.log("Field is not a struct or pointer... continuing until this is supported")

			continue
		}

		dependencyType := ij.tr.FindTypeByValue(field)

		if objIsPointer(dependencyType) {
			ij.log(fmt.Sprintf("Type: %s, (a pointer) found in type registry", reflect.TypeOf(dependencyType).Elem().Name()))
		}

		ij.log("Provisioning instance of: " + reflect.TypeOf(dependencyType).Elem().String())

		// OKAY NOW LOOK FOR THIS IN REGISTERED OBJS
		foundDep := ij.tr.FindBuiltObjectByValue(reflect.ValueOf(dependencyType))

		if foundDep != nil {
			// @todo works if asking for a pointer, not if asking for an elem
			if field.Kind() == reflect.Ptr {
				field.Set(reflect.ValueOf(foundDep).Elem().Elem())
			} else {
				field.Set(reflect.ValueOf(foundDep).Elem().Elem().Elem())
			}

			continue
		}

		// Works to give us a pointer... BUT what if field asks for a struct???
		concreteDependency := reflect.Indirect(reflect.ValueOf(dependencyType)).Interface()

		// OKAY, if field is asking for a pointer, then set pointer.
		if field.Kind() == reflect.Ptr {
			// Convert again.Needthis to *again.Needthis
			concreteDepInterface := reflect.New(reflect.TypeOf(concreteDependency))
			concreteDepInterface.Elem().Set(reflect.ValueOf(concreteDependency))

			field.Set(concreteDepInterface)
		} else {
			// field is asking for a concrete
			copyObj := reflect.New(reflect.TypeOf(concreteDependency))
			copyObj.Elem().Set(reflect.ValueOf(concreteDependency))
			copyObj = copyObj.Elem()

			field.Set(copyObj)
		}
	}

	return concreteObj
}

/* Log normal 'debug-level' stuff. */
func (ij *Injector) log(msg string) {
	if ij.logger != nil {
		ij.logger.Debug(msg)
	}
}

/* Log error stuff. */
func (ij *Injector) elog(msg string) {
	if ij.logger != nil {
		ij.logger.Error(msg)
	}
}
