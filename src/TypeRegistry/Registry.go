package TypeRegistry

/* The types available in the auto-generated registry. */
type RegistryStruct struct {
	Name           string
	Implementation interface{}
}

type RegistryInterface RegistryStruct

type RegistryFactory struct {
	Name            string
	Implementations []interface{}
}

type Registry struct {
	RegistryStructs    []RegistryStruct
	RegistryFactories  []RegistryFactory
	RegistryInterfaces []RegistryInterface
}
