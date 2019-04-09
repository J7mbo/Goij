package TypeRegistry

import (
	"fmt"
	"reflect"
	"strings"
)

/* A structRegistry containing all structs, and their package names, in the application. */
type TypeRegistry struct {
	/* Only contains struct types. */
	structRegistry map[string]interface{}

	/* Only contains interface types. */
	interfaceRegistry map[string]interface{}

	/* Only contains factories. */
	factoryRegistry map[string][]interface{}
}

/* Terrible wizardry. You can pass the registry in created from having run ./bin/gen. */
func New(registries ...Registry) *TypeRegistry {
	structRegistry := make(map[string]interface{})
	interfaceRegistry := make(map[string]interface{})
	factoryRegistry := make(map[string][]interface{})

	for _, userRegistry := range registries {
		for _, registryStruct := range userRegistry.RegistryStructs {
			structRegistry[registryStruct.Name] = registryStruct.Implementation
		}

		for _, registryInterface := range userRegistry.RegistryInterfaces {
			interfaceRegistry[registryInterface.Name] = registryInterface.Implementation
		}

		for _, registryFactory := range userRegistry.RegistryFactories {
			for _, implementation := range registryFactory.Implementations {
				factoryRegistry[registryFactory.Name] = append(factoryRegistry[registryFactory.Name], implementation)
			}
		}
	}

	return &TypeRegistry{
		structRegistry:    structRegistry,
		interfaceRegistry: interfaceRegistry,
		factoryRegistry:   factoryRegistry,
	}
}

func (r *TypeRegistry) Add(registry Registry) {
	for _, registryStruct := range registry.RegistryStructs {
		r.structRegistry[registryStruct.Name] = registryStruct.Implementation
	}

	for _, registryInterface := range registry.RegistryInterfaces {
		r.interfaceRegistry[registryInterface.Name] = registryInterface.Implementation
	}

	for _, registryFactory := range registry.RegistryFactories {
		for _, implementation := range registryFactory.Implementations {
			r.factoryRegistry[registryFactory.Name] = append(r.factoryRegistry[registryFactory.Name], implementation)
		}
	}
}

func (r *TypeRegistry) FindStructType(name string) interface{} {
	/* Is this the short name? If so, try and match on a single struct. */
	if !strings.Contains(name, ".") {
		found := make([]interface{}, 0)

		for registryStructFQName, registryStruct := range r.structRegistry {
			split := strings.Split(registryStructFQName, ".")

			/* The registry contains something invalid then - probably user added. */
			if len(split) == 0 {
				continue
			}

			if split[len(split)-1] == name {
				found = append(found, registryStruct)
			}
		}

		if len(found) == 1 {
			return found[0]
		}
	}

	if theType, exists := r.structRegistry[name]; exists {
		return theType
	}

	return nil
}

func (r *TypeRegistry) FindInterfaceType(name string) interface{} {
	/* Is this the short name? If so, try and match on a single struct. */
	if !strings.Contains(name, ".") {
		found := make([]interface{}, 0)

		for registryInterfaceFQName, registryInterface := range r.interfaceRegistry {
			split := strings.Split(registryInterfaceFQName, ".")

			/* The registry contains something invalid then - probably user added. */
			if len(split) == 0 {
				continue
			}

			if split[len(split)-1] == name {
				found = append(found, reflect.TypeOf(registryInterface).Elem())
			}
		}

		if len(found) == 1 {
			return found[0]
		}
	}

	if theType, exists := r.interfaceRegistry[name]; exists {
		return reflect.TypeOf(theType).Elem()
	}

	return nil
}

func (r *TypeRegistry) FindFactoryTypes(name string) []interface{} {
	if theType, exists := r.factoryRegistry[name]; exists {
		return theType
	}

	return nil
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *TypeRegistry) FindInterfaceTypeByType(objType reflect.Type) interface{} {
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	if obj := r.FindInterfaceType(fmt.Sprintf("%s.%s", objType.PkgPath(), objType.Name())); obj != nil {
		return toStructPointer(obj)
	}

	return nil
}

/* Given an interface name, if registered, return all struct types that implement it. */
func (r *TypeRegistry) FindStructTypesByInterfaceType(interfaceName string) (structs []interface{}) {
	/* Is this the short name? If so, try and match on a single interface. */
	if !strings.Contains(interfaceName, ".") {
		found := make([]interface{}, 0)

		for registryInterfaceFQName, registryInterface := range r.interfaceRegistry {
			split := strings.Split(registryInterfaceFQName, ".")

			/* The registry contains something invalid then - probably user added. */
			if len(split) == 0 {
				continue
			}

			if split[len(split)-1] == interfaceName {
				found = append(found, registryInterface)
			}
		}

		if len(found) != 1 {
			/* Can't use short name then. */
			return structs
		}

		fullName := fmt.Sprintf("%s.%s", reflect.TypeOf(found[0]).PkgPath(), reflect.TypeOf(found[0]).Name())

		interfaceName = fullName
	}

	interfaceType := r.FindInterfaceType(interfaceName)

	if interfaceType == nil {
		return structs
	}

	for _, structType := range r.structRegistry {
		/* This cast assumes a non-fucked up interface type registry. */
		if reflect.PtrTo(reflect.TypeOf(structType)).Implements(interfaceType.(reflect.Type)) {
			structs = append(structs, structType)
		}
	}

	return structs
}

/* Given a reflect value (when recursing around a struct's fields with reflect); find the object already stored. */
func (r *TypeRegistry) FindStructTypeByType(objType reflect.Type) interface{} {
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	if obj := r.FindStructType(fmt.Sprintf("%s.%s", objType.PkgPath(), objType.Name())); obj != nil {
		return toStructPointer(obj)
	}

	return nil
}

/* Given a struct, create a pointer to it so we can reflect / edit values on it. */
func toStructPointer(obj interface{}) interface{} {
	vp := reflect.New(reflect.TypeOf(obj))
	vp.Elem().Set(reflect.ValueOf(obj))

	return vp.Interface()
}
