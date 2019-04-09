/* Package Goij provides a fully automatic recursive dependency injector for dependency initialisation and injection. */
package Goij

import (
	"fmt"
	"github.com/j7mbo/goij/src/Cache"
	"github.com/j7mbo/goij/src/Logger"
	"github.com/j7mbo/goij/src/TypeRegistry"
	"reflect"
	"strings"
)

/* Injector is the interface returned from calling NewInjector() and contains the methods for dependency initialisation. */
type Injector interface {
	/*
		Make initialises a struct type from a string as long as the string is a key in the TypeRegistry.

		Make also recursively initialises and injects all dependencies for all public structs into a copy of the found type.
		An interface name can also be provided and a concrete implementation will be attempted to be initialised.
	*/
	Make(name string) interface{}

	/*
		Share enables the sharing of a struct for any future injection usage.

		Create the object once, share it, and it will be injected first when encountered in any recursive call.
	*/
	Share(object interface{})

	/*
		Bind binds an interface to a struct implementation for any future injection usage.

		Any encounter of the interface in any future recursive call will have the implementation injected in it's place.
		Bind must be used whenever multiple implementing types exist in the type registry for a single interface.
	*/
	Bind(interfaceName string, structName string)

	/*
		Delegate delegates the initialisation of a struct type to a lambda or first class function type.

		Any encounter of the struct type in any future recursive calls will have the factory initialise the struct.
		The factory does not cache the resulting object due to the possibility that it's contents is dynamic.
	*/
	Delegate(structName string, factoryMethod interface{})

	/*
		Define allows injection definitions for specific objects.
	*/
	Define(structName string, paramName string, value interface{})

	/*
		DefineGlobal allows the global definition of scalars such as strings, integers etc to be injected everywhere.

		The matching is performed on property name.
	*/
	DefineGlobal(paramName string, value interface{})

	/*
		Invoke executes a function on the given object and returns all return values as an array.
	*/
	Invoke(object interface{}, methodName string, args ...interface{}) []interface{}
}

type injector struct {
	/* Registry of all application types. */
	tr *TypeRegistry.TypeRegistry

	/* Contains any cached objects we want to draw from. */
	objectCache Cache.ObjectCache

	/* Initialisation delegates (factories). */
	delegates Cache.DelegateCache

	/* Optional if you want to know what wizardry is occurring. */
	logger *Logger.Logger

	/* Bindings from interface to concrete. */
	bindings map[string]string

	/* Scalar parameter definitions. */
	definitions map[string]map[string]interface{}

	/* Global scalar parameter definitions. */
	globalDefinitions map[string]interface{}
}

func NewInjector(tr *TypeRegistry.TypeRegistry, logger *Logger.Logger) Injector {
	return &injector{
		tr:                tr,
		logger:            logger,
		objectCache:       Cache.NewObjectCache(),
		delegates:         Cache.NewDelegateCache(),
		definitions:       make(map[string]map[string]interface{}),
		globalDefinitions: make(map[string]interface{}),
	}
}

/* Format: PackageName.StructName. */
func (ij *injector) Make(name string) interface{} {
	ij.log(fmt.Sprintf("injector asked to provision: '%s' by user", name))

	/* Let's check the struct and interface registries. */
	obj := ij.getObjFromStructOrInterfaceTypeRegistry(name)

	/* See if this object is already cached? */
	foundObj := ij.objectCache.FindByValue(reflect.ValueOf(obj))

	if foundObj != nil {
		ij.log(fmt.Sprintf("Object of type: '%T' was already provisioned in registry - returning.", getValue(foundObj)))

		return toStructPtr(getValue(foundObj))
	}

	delegateOrFactory := ij.findAndCallDelegateOrFactory(obj)

	if delegateOrFactory != nil {
		return delegateOrFactory
	}

	/* Provision all child fields of this top level object. */
	builtObj := ij.buildFields(obj, obj)

	/* Cache the object. */
	ij.objectCache.Store(toStructPtr(getValue(builtObj)))

	return builtObj
}

/* Define scalar parameters for injection. */
func (ij *injector) Define(objectName string, paramName string, value interface{}) {
	if _, found := ij.definitions[objectName]; !found {
		ij.definitions[objectName] = make(map[string]interface{})
	}

	ij.definitions[objectName][paramName] = value
}

/* Define global scalar parameters for injection. */
func (ij *injector) DefineGlobal(paramName string, value interface{}) {
	ij.globalDefinitions[paramName] = value
}

/* Delegate the initialisation of an object to a factory method. */
func (ij *injector) Delegate(objectName string, factoryMethod interface{}) {
	ij.delegates.Store(objectName, factoryMethod)
}

func (ij *injector) Bind(interfaceName string, structName string) {
	if ij.bindings == nil {
		ij.bindings = make(map[string]string)
	}

	var interfaceType interface{}
	var structType interface{}

	if interfaceType = ij.tr.FindInterfaceType(interfaceName); interfaceType == nil {
		panic(fmt.Sprintf("Interface type: '%s' not found in struct registry, did you register it?", interfaceName))
	}

	if structType = ij.tr.FindStructType(structName); structType == nil {
		panic(fmt.Sprintf("RegistryStruct type: '%s' not found in struct registry, did you register it?", structName))
	}

	ij.bindings[interfaceName] = structName
}

func (ij *injector) Invoke(object interface{}, methodName string, args ...interface{}) []interface{} {
	inputs := make([]reflect.Value, len(args))

	for i, arg := range args {
		inputs[i] = reflect.ValueOf(arg)
	}

	results := reflect.ValueOf(object).MethodByName(methodName).Call(inputs)

	outputs := make([]interface{}, len(results))

	for i, result := range results {
		outputs[i] = result.Interface()
	}

	return outputs
}

func (ij *injector) Share(obj interface{}) {
	ij.objectCache.Store(obj)
}

/* Checks both the struct registry and the interface registry. */
func (ij *injector) getObjFromStructOrInterfaceTypeRegistry(name string) interface{} {
	obj := ij.tr.FindStructType(name)

	if obj != nil {
		/* Object in registry is a struct - so create a ptr copy so when we pass obj in, it is updated recursively. */
		return toStructPtr(obj)
	}

	/* Is it an interface though? */
	interfaceType := ij.tr.FindInterfaceType(name)

	if interfaceType == nil {
		ij.panic(fmt.Sprintf("No type found in registry for name: '%s', did you forget to register it?", name))
	}

	/* Is the interface bound to a single concrete type via bind()? */
	if structName, found := ij.bindings[name]; found {
		return toStructPtr(ij.tr.FindStructType(structName))
	}

	/* Does the interface have a delegate (for when there are no exported structs for that interface)? */
	delegateResults := ij.findAndCallDelegateOrFactory(interfaceType)

	if delegateResults != nil {
		return delegateResults
	}

	/* Interface type exists so search for a single implementing type. If more, user needs to bind one. */
	structTypes := ij.tr.FindStructTypesByInterfaceType(name)

	switch lenStructs := len(structTypes); {
	case lenStructs == 0:
		ij.panic("You can't Make() an interface unless there is exactly one implementing type in the registry.")
	case lenStructs > 1:
		ij.panic(
			fmt.Sprintf(
				"Multiple implementing types were found for interface: '%s', specify one with bind()", name,
			),
		)
	default:
		obj = structTypes[0]

		ij.log(
			fmt.Sprintf("Single object of type: '%T' implementing: '%s' was found and provisioned", obj, name),
		)
	}

	return toStructPtr(obj)
}

func (ij *injector) buildFields(topLevelObj interface{}, parentObj interface{}) interface{} {
	value, fieldCount := ij.getValueAndNumFields(parentObj)

	if fieldCount == 0 {
		return topLevelObj
	}

	for i := 0; i < fieldCount; i++ {
		fieldName := reflect.TypeOf(getValue(parentObj)).Field(i).Name
		fieldType := reflect.TypeOf(getValue(parentObj)).Field(i).Type
		fieldIsPointer := reflect.TypeOf(getValue(parentObj)).Field(i).Type.Kind() == reflect.Ptr
		valueIsPointer := value.Elem().Kind() == reflect.Ptr

		var field interface{}

		if valueIsPointer {
			/* Ignore private fields */
			if !value.Elem().Elem().Field(i).CanSet() {
				ij.log(
					fmt.Sprintf(
						"Found private %s field: %s of type: %s on object: %T, ignoring...",
						fieldType.Kind(), fieldName, fieldType, parentObj,
					),
				)

				continue
			}

			/* Use Addr() to get the actually 'settable' field. */
			field = value.Elem().Elem().Field(i).Addr().Interface()
		} else {
			/* Ignore private fields */
			if !value.Elem().Field(i).CanInterface() {
				ij.log(
					fmt.Sprintf(
						"Found private %s field: %s of type: %s on object: %T, ignoring...",
						fieldType.Kind(), fieldName, fieldType, parentObj,
					),
				)

				continue
			}

			field = value.Elem().Field(i).Addr().Interface()
		}

		ij.log(
			fmt.Sprintf(
				"Found %s field: %s of type: %s on object: %T", fieldType.Kind(), fieldName, fieldType, parentObj,
			),
		)

		/* Interfaces */
		if fieldType.Kind() == reflect.Interface {
			obj := ij.provisionTypeFromInterface(fieldType, fieldName)

			/* We found a single or bound type, great... but do we have this single or bound type already cached? */
			dep := ij.objectCache.FindByType(reflect.TypeOf(obj))

			if dep != nil {
				ij.log(
					fmt.Sprintf(
						"Dependency of type: '%T' was already provisioned in registry - returning.", getValue(dep),
					),
				)

				getElem(value.Interface()).Field(i).Set(reflect.ValueOf(toStructPtr(getValue(dep))))

				/* Cache the dependency now - it wasn't created by a factory so it's okay to cache it. */
				ij.objectCache.Store(dep)

				continue
			}

			/* Any user-registered delegates or automatic factories available for it? */
			delegateOrFactoryResult := ij.findAndCallDelegateOrFactory(obj)

			if delegateOrFactoryResult != nil {
				ij.log(fmt.Sprintf("Found delegate for type: %T. Delegate called and returned: %T", obj, delegateOrFactoryResult))

				obj = delegateOrFactoryResult
			}

			/* Okay, are there any factories available for the INTERFACE instead? */
			if delegateOrFactoryResult == nil {
				// @todo changed this from fieldType to field, does it work?
				delegateOrFactoryResult = ij.findAndCallDelegateOrFactory(field)

				if delegateOrFactoryResult != nil {
					ij.log(fmt.Sprintf("Found delegate for type: %T. Delegate called and returned: %T", obj, delegateOrFactoryResult))

					obj = delegateOrFactoryResult
				}
			}

			obj = toStructPtr(obj)

			getElem(value.Interface()).Field(i).Set(reflect.ValueOf(toStructPtr(getValue(obj))))

			ij.buildFields(topLevelObj, obj)

			continue
		}

		/* Scalars */
		if !fieldIsPointer && fieldType.Kind() != reflect.Struct || (fieldIsPointer && fieldType.Elem().Kind() != reflect.Struct) {
			foundDefinition := ij.findDefinitionOrGlobalDefinition(value, fieldName)

			if foundDefinition != nil {
				getElem(value.Interface()).Field(i).Set(reflect.ValueOf(foundDefinition))

				continue
			}

			/* We don't want to recurse with buildFields for user-provided definitions. */
			continue
		}

		/* If the user has defined a specific injection definition, use this... comes first so overrides Share(). */
		foundDefinition := ij.findDefinitionOrGlobalDefinition(value, fieldName)

		if foundDefinition != nil {
			ij.log(
				fmt.Sprintf(
					"Definition of type: '%T' was found for object: %T - injecting.", foundDefinition, value,
				),
			)

			getElem(value.Interface()).Field(i).Set(reflect.ValueOf(foundDefinition))

			/* We don't want to recurse with buildFields for user-provided definitions. */
			continue
		}

		/* Has the object already been cached by the user? */
		dep := ij.objectCache.FindByType(fieldType)

		if dep != nil {
			ij.log(
				fmt.Sprintf(
					"Dependency of type: '%T' was already provisioned in registry - returning.", getValue(dep),
				),
			)

			if fieldIsPointer {
				getElem(value.Interface()).Field(i).Set(reflect.ValueOf(toStructPtr(getValue(dep))))
			} else {
				getElem(value.Interface()).Field(i).Set(reflect.ValueOf(getValue(dep)))
			}

			/* Cache the dependency now - we don't want to cache factory results below as they may be dynamic. */
			ij.objectCache.Store(dep)

			continue
		}

		delegateOrFactory := ij.findAndCallDelegateOrFactory(fieldType)

		if delegateOrFactory != nil {
			dep = delegateOrFactory
		} else {
			/* Object has not been cached by the user nor is there a factory for it - initialise. */
			dep = ij.tr.FindStructTypeByType(fieldType)
		}

		if dep == nil {
			ij.panic(fmt.Sprintf("No type found in registry for name: '%s', did you forget to register it?", fieldName))
		}

		if fieldIsPointer {
			getElem(value.Interface()).Field(i).Set(reflect.ValueOf(toStructPtr(getElem(dep).Interface())))
		} else {
			getElem(value.Interface()).Field(i).Set(getElem(dep))
		}

		/* If a factory has returned an object, we don't need to recurse on it as the user has decided to build it. */
		if delegateOrFactory == nil {
			ij.buildFields(topLevelObj, field)
		}
	}

	return topLevelObj
}

/* On encountering a field asking for an interface, try and figure out which struct to inject. */
func (ij *injector) provisionTypeFromInterface(fieldType reflect.Type, fieldName string) interface{} {
	interfaceType := ij.tr.FindInterfaceTypeByType(fieldType)

	if interfaceType == nil {
		ij.panic(
			fmt.Sprintf(
				"No interface found in registry for name: '%s', did you forget to register it?", fieldName,
			),
		)
	}

	/* We know it exists in the registry now. */
	fullInterfaceName := fieldType.PkgPath() + "." + fieldType.Name()

	/* Interface type exists so search for a single implementing type. If more exist, user needs to bind one. */
	structTypes := ij.tr.FindStructTypesByInterfaceType(fullInterfaceName)

	var obj interface{}

	/* Is the interface bound to a single concrete type via bind()? */
	if structName, found := ij.bindings[fullInterfaceName]; found {
		return ij.tr.FindStructType(structName)
	}

	/* What about a short name for the interface? */
	if structName, found := ij.bindings[fieldType.Name()]; found {
		return ij.tr.FindStructType(structName)
	}

	/* Does the interface have a delegate (for when there are no exported structs for that interface)? */
	delegateOrFactoryResult := ij.findAndCallDelegateOrFactory(interfaceType)

	if delegateOrFactoryResult != nil {
		return delegateOrFactoryResult
	}

	switch lenStructs := len(structTypes); {
	case lenStructs == 0:
		ij.panic(
			"Could not initialise interface dependency unless there is exactly one implementing type in " +
				"the registry or it has been bound to a single type with bind().",
		)
	case lenStructs > 1:
		ij.panic(
			fmt.Sprintf(
				"Multiple implementing types were found for interface: '%s', specify one with bind()",
				fullInterfaceName,
			),
		)
	default:
		obj = toStructPtr(structTypes[0])

		ij.log(
			fmt.Sprintf(
				"Found single mapping of: '%T' implementing: '%s', provisioning", obj, fullInterfaceName,
			),
		)
	}

	return obj
}

func (ij *injector) findDefinitionOrGlobalDefinition(value reflect.Value, fieldName string) interface{} {
	/* Is there a short name available (without the package path, so "testObject"). ? */
	shortName := value.Type().Elem().Name()

	if definition, found := ij.definitions[shortName]; found {
		if definitionVal, found := definition[fieldName]; found {
			return definitionVal
		}
	}

	/* How about the long name? */
	var valueName string

	valueIsPointer := value.Elem().Kind() == reflect.Ptr

	if valueIsPointer {
		valueName = fmt.Sprintf("%s.%s", value.Type().PkgPath(), value.Type().Name())
	} else {
		valueName = fmt.Sprintf("%s.%s", value.Type().Elem().PkgPath(), value.Type().Elem().Name())
	}

	if definition, found := ij.definitions[valueName]; found {
		if definitionVal, found := definition[fieldName]; found {
			return definitionVal
		}
	}

	/* Is there a globally available injection definition?  */
	if definitionVal, found := ij.globalDefinitions[fieldName]; found {
		return definitionVal
	}

	return nil
}

func (ij *injector) findAndCallDelegateOrFactory(objType interface{}) interface{} {
	/* Any user-registered delegates for it? */
	userProvidedDelegate := ij.delegates.FindByType(reflect.TypeOf(objType))

	if userProvidedDelegate != nil {
		return ij.callDelegate(userProvidedDelegate)
	}

	/* Automatic factory usage possible? */
	typeName := fmt.Sprintf("%s.%s", getElem(objType).Type().PkgPath(), getElem(objType).Type().Name())

	/* Did we pass in a reflect.Type? Easier than checking the type of the type of the type etc.. */
	if typeName == "reflect.rtype" {
		/* We can have a **reflect.rtype, don't ask me why. I lost that a long time ago in this craziness. */
		objType = getElem(objType).Addr().Interface()

		assertedType := objType.(reflect.Type)

		typeName = fmt.Sprintf("%s.%s", assertedType.PkgPath(), assertedType.Name())

		/*
			If this is an interface, but the type is private, Name() will be lowercased so won't be found:

			ie: Injector interface, but here assertedType.Name() will be "injector"..
		*/

		userProvidedDelegate = ij.delegates.FindByName(typeName)

		if userProvidedDelegate != nil {
			return ij.callDelegate(userProvidedDelegate)
		}

		/* What about user-provided short-names? */
		assertedTypeName := assertedType.Name()

		userProvidedDelegate = ij.delegates.FindByName(assertedTypeName)

		if userProvidedDelegate != nil {
			return ij.callDelegate(userProvidedDelegate)
		}
	}

	/* What about user-provided short-names? */
	userProvidedDelegate = ij.delegates.FindByName(getElem(objType).Type().Name())

	if userProvidedDelegate != nil {
		return ij.callDelegate(userProvidedDelegate)
	}

	factoryDelegate := ij.getFactoryFromFactoryRegistry(typeName)

	if factoryDelegate != nil {
		ij.log(fmt.Sprintf("Found single factory delegate automatically in registry: %T", factoryDelegate))

		args := ij.resolveInvocationArgs(factoryDelegate)

		if len(args) > 0 {
			ij.log(fmt.Sprintf("Ready to inject args: %v into factory delegate: %T", args, factoryDelegate))
		}

		factoryReturns := reflect.ValueOf(factoryDelegate).Call(args)

		return factoryReturns[0].Interface()
	}

	return nil
}

func (ij *injector) callDelegate(delegate interface{}) interface{} {
	factoryReturns := reflect.ValueOf(delegate).Elem().Call(
		ij.resolveInvocationArgs(delegate),
	)

	return factoryReturns[0].Interface()
}

/* Resolves the invocation args for a provided function type. */
func (ij *injector) resolveInvocationArgs(object interface{}) (results []reflect.Value) {
	var objectType reflect.Type

	if reflect.TypeOf(object).Kind() == reflect.Ptr {
		objectType = reflect.TypeOf(object).Elem()
	} else {
		objectType = reflect.TypeOf(object)
	}

	if objectType.Kind() != reflect.Func {
		return nil
	}

	numArguments := objectType.NumIn()

	if numArguments == 0 {
		ij.log(fmt.Sprintf("No invocation args required for delegate: %T", object))

		return nil
	}

	ij.log(fmt.Sprintf("Resolving invocation args for delegate: %T", object))

	for i := 0; i < numArguments; i++ {
		arg := objectType.In(i)

		/* Argument names cannot be retrieved with reflection for functions, so they must be the zero value instead. */
		if (arg.Kind() != reflect.Interface && arg.Kind() != reflect.Struct && arg.Kind() != reflect.Ptr) ||
			(arg.Kind() == reflect.Ptr && arg.Elem().Kind() != reflect.Interface && arg.Elem().Kind() != reflect.Struct) {
			ij.log(
				fmt.Sprintf(
					"Encountered scalar delegate argument: %T for delegate: %T, injecting zero value", arg, object,
				),
			)

			/* In the case it's a pointer to a scalar... like *int64... */
			if arg.Kind() == reflect.Ptr && arg.Elem().Kind() != reflect.Struct {
				results = append(results, reflect.New(arg.Elem()))
			} else {
				results = append(results, reflect.New(arg).Elem())
			}

			continue
		}

		/* If interface - resolve interface to struct first.. */
		if arg.Kind() == reflect.Interface {
			argFQName := fmt.Sprintf("%s.%s", arg.PkgPath(), arg.Name())

			ij.log(fmt.Sprintf("Encountered interface delegate argument: %s for delegate: %T", argFQName, object))

			/* Check if there is a delegate specifically for this interface first... */
			if delegateOrFactoryResult := ij.findAndCallDelegateOrFactory(arg); delegateOrFactoryResult != nil {
				ij.log(
					fmt.Sprintf(
						"Retrieved delegate or factory result: %T, for delegate interface argument: %v, for delegate: %T",
						delegateOrFactoryResult, arg, object,
					),
				)

				// @todo - Depending on pointer or not??

				results = append(results, reflect.ValueOf(delegateOrFactoryResult))

				continue
			}

			if resolvedStruct := ij.provisionTypeFromInterface(arg, argFQName); resolvedStruct != nil {
				/* Found struct type from type registry - replace interface in arg var and continue. */
				if reflect.TypeOf(resolvedStruct).Kind() == reflect.Ptr && arg.Kind() != reflect.Ptr {
					arg = reflect.TypeOf(resolvedStruct).Elem()
				} else {
					arg = reflect.TypeOf(resolvedStruct)
				}

				/*
					If the argument is the same as the return type from a delegate, it'll be infinitely recursive so avoid..

					Naively assumes factories only return one object of the type we want...
				*/
				returnValue := objectType.Out(0)

				if strings.ToLower(arg.String()) == strings.ToLower(returnValue.String()) {
					results = append(results, reflect.ValueOf(resolvedStruct))

					continue
				}
			}
		}

		/* Use cached arg if one exists.. */
		if obj := ij.objectCache.FindByType(arg); obj != nil {
			ij.log(fmt.Sprintf("Encountered cached delegate argument: %T for delegate: %T", obj, object))

			if arg.Kind() == reflect.Ptr && reflect.TypeOf(obj).Elem().Kind() != reflect.Ptr {
				results = append(results, reflect.ValueOf(obj))

				continue
			}

			/* Cached things look like **elem, and the delegate arg is not a pointer. */
			if arg.Kind() == reflect.Struct && reflect.TypeOf(obj).Elem().Kind() == reflect.Ptr {
				results = append(results, getElem(obj))

				continue
			}

			results = append(results, reflect.ValueOf(obj).Elem())

			continue
		}

		/* User delegate or factory? This is effectively a recursive call... */
		if delegateOrFactoryResult := ij.findAndCallDelegateOrFactory(arg); delegateOrFactoryResult != nil {
			/* In the case that the argument is an interface but we have a struct... */
			if arg.Kind() == reflect.Interface && reflect.TypeOf(delegateOrFactoryResult).Kind() == reflect.Struct {
				delegateOrFactoryResult = reflect.ValueOf(reflect.PtrTo(reflect.TypeOf(delegateOrFactoryResult))).Interface()
			} else if objectType.In(i).Kind() == reflect.Struct && reflect.TypeOf(delegateOrFactoryResult).Kind() == reflect.Ptr {
				delegateOrFactoryResult = reflect.ValueOf(delegateOrFactoryResult).Elem().Interface()
			}

			ij.log(
				fmt.Sprintf(
					"Retrieved delegate or factory result: %T, for delegate argument: %v, for delegate: %T",
					delegateOrFactoryResult, arg, object,
				),
			)

			results = append(results, reflect.ValueOf(delegateOrFactoryResult))

			continue
		}

		var newArg interface{}

		if arg.Kind() == reflect.Ptr {
			newArg = reflect.New(arg.Elem()).Interface()
		} else {
			newArg = reflect.New(arg).Interface()
		}

		if objectType.In(i).Kind() == reflect.Struct && reflect.TypeOf(newArg).Kind() == reflect.Ptr {
			newArg = reflect.ValueOf(newArg).Elem().Interface()
		}

		ij.log(fmt.Sprintf("Provisioning new argument: %T (as none cached) for delegate: %T", newArg, object))

		results = append(results, reflect.ValueOf(ij.buildFields(newArg, newArg)))
	}

	return
}

/*
The only reason we would be calling this method is if there was not a factory delegated already, so this is for auto
factory usage only.
*/
func (ij *injector) getFactoryFromFactoryRegistry(name string) interface{} {
	factoryTypes := ij.tr.FindFactoryTypes(name)

	numFactories := len(factoryTypes)

	if numFactories == 1 {
		return factoryTypes[0]
	}

	if numFactories > 1 {
		ij.panic(
			fmt.Sprintf(
				"More than one factory exists in registry for object: '%s', you must Delegate() one first", name,
			),
		)
	}

	return nil
}

func (ij *injector) getValueAndNumFields(obj interface{}) (reflect.Value, int) {
	val := reflect.ValueOf(&obj)
	num := getElem(obj).NumField()

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	ij.log(fmt.Sprintf("Object: '%s' has %d field(s)", reflect.TypeOf(val.Interface()).String(), num))

	return reflect.ValueOf(toStructPtr(val.Interface())).Elem(), num
}

/* Given a value, loop through until we get a concrete element out of it. */
func getValue(obj interface{}) interface{} {
	return getElem(obj).Interface()
}

func getElem(obj interface{}) reflect.Value {
	val := reflect.ValueOf(obj)

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val
}

func toStructPtr(obj interface{}) interface{} {
	vp := reflect.New(reflect.TypeOf(obj))
	vp.Elem().Set(reflect.ValueOf(obj))

	return vp.Interface()
}

/* Log normal 'debug-level' stuff. */
func (ij *injector) log(msg string) {
	if ij.logger != nil {
		ij.logger.Debug(msg)
	}
}

/* Log error stuff. */
func (ij *injector) elog(msg string) {
	if ij.logger != nil {
		ij.logger.Error(msg)
	}
}

/* Log imminent death. And then die. */
func (ij *injector) panic(msg string) {
	ij.elog(msg)

	panic(msg)
}
