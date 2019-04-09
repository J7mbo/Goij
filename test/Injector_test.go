package test

import (
	"github.com/j7mbo/MethodCallRetrier"
	"github.com/j7mbo/goij"
	"github.com/j7mbo/goij/src/Logger"
	"github.com/j7mbo/goij/src/TypeRegistry"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"testing"
)

type InjectorTestSuite struct {
	suite.Suite
}

func TestInjectorTestSuite(t *testing.T) {
	tests := new(InjectorTestSuite)

	suite.Run(t, tests)
}

func (s *InjectorTestSuite) TestCanMakeSimpleObject() {
	x := struct{}{}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.x", Implementation: x},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&x, ij.Make("github.com/j7mbo/goij/test.x"))
}

func (s *InjectorTestSuite) TestCanMakeSimpleObjectWithSiblingDependencies() {
	type Dep struct{}
	type DepTwo struct{}
	type DepThree struct{}

	dep := Dep{}
	dep2 := DepTwo{}
	dep3 := DepThree{}

	obj := struct {
		Dep        Dep
		AnotherDep DepTwo
		ThirdDep   DepThree
	}{
		Dep:        dep,
		AnotherDep: dep2,
		ThirdDep:   dep3,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: obj},
			{Name: "github.com/j7mbo/goij/test.Dep", Implementation: dep},
			{Name: "github.com/j7mbo/goij/test.DepTwo", Implementation: dep2},
			{Name: "github.com/j7mbo/goij/test.DepThree", Implementation: dep3},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&obj, ij.Make("github.com/j7mbo/goij/test.Obj"))
}

func (s *InjectorTestSuite) TestCanMakeSimpleObjectWithLargeDependencyTree() {
	type DepFour struct{}
	type DepThree struct {
		DepFour DepFour
	}
	type DepTwo struct {
		DepThree DepThree
	}
	type Dep struct {
		DepTwo *DepTwo
	}

	dep4 := DepFour{}
	dep3 := DepThree{
		DepFour: dep4,
	}
	dep2 := DepTwo{
		DepThree: dep3,
	}
	dep := Dep{
		DepTwo: &dep2,
	}

	type Obj struct {
		Dep Dep
	}

	obj := Obj{Dep: dep}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: obj},
			{Name: "github.com/j7mbo/goij/test.Dep", Implementation: dep},
			{Name: "github.com/j7mbo/goij/test.DepTwo", Implementation: dep2},
			{Name: "github.com/j7mbo/goij/test.DepThree", Implementation: dep3},
			{Name: "github.com/j7mbo/goij/test.DepFour", Implementation: dep4},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&obj, ij.Make("github.com/j7mbo/goij/test.Obj"))
}

func (s *InjectorTestSuite) TestCanMakeSimpleObjectWithPointerDependency() {
	type Dep struct{}
	dep := &Dep{}

	obj := struct {
		Dep *Dep
	}{
		Dep: dep,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: obj},
			{Name: "github.com/j7mbo/goij/test.Dep", Implementation: dep},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&obj, ij.Make("github.com/j7mbo/goij/test.Obj"))
}

func (s *InjectorTestSuite) TestMakeObjectUpdatedDoesNotAffectCachedObject() {
	x := struct {
		Int int
	}{
		Int: 42,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.x", Implementation: x},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	makeX := ij.Make("github.com/j7mbo/goij/test.x")
	makeX.(*struct{ Int int }).Int = 21

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.x").(*struct{ Int int }).Int)
}

func (s *InjectorTestSuite) TestMakeIgnoresPrivateFieldsAndPointerPrivateFieldsAreSetToNil() {
	type dep struct{}

	x := struct {
		Int int
		dep *dep
	}{}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.x", Implementation: x},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Nil(ij.Make("github.com/j7mbo/goij/test.x").(*struct {
		Int int
		dep *dep
	}).dep)
}

/* Mainly for coverage on the valueIsPointer check in Injector.go. */
func (s *InjectorTestSuite) TestPrivateChildDependencyOfPointerDependencyIsIgnored() {
	type Dep2 struct{}
	type Dep struct {
		dep2 *Dep2
	}
	type Obj struct {
		Dep *Dep
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: Obj{}},
			{Name: "github.com/j7mbo/goij/test.Dep", Implementation: Dep{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Nil(ij.Make("github.com/j7mbo/goij/test.Obj").(*Obj).Dep.dep2)
}

func (s *InjectorTestSuite) TestMakeIgnoresPrivateFieldsAndStructPrivateFieldsAreSetToZeroedValue() {
	type depsDep struct{}

	type dep struct {
		DepsDep depsDep
	}

	type obj struct {
		Int int
		dep dep
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.x", Implementation: obj{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Empty(ij.Make("github.com/j7mbo/goij/test.x").(*obj).dep.DepsDep)
}

func (s *InjectorTestSuite) TestMakeOnObjectNotInRegistryPanics() {
	/* Yes, for test coverage -.-. */
	nullLogger := Logger.New(
		func(...interface{}) {},
		func(...interface{}) {},
	)

	ij := Goij.NewInjector(TypeRegistry.New(), &nullLogger)

	s.Assert().Panics(func() {
		ij.Make("doesnt.exist")
	})
}

func (s *InjectorTestSuite) TestCanShareDefinedObject() {
	type X struct{}

	x := X{}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.x", Implementation: x},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Share(x)

	s.Assert().Equal(&x, ij.Make("github.com/j7mbo/goij/test.x"))
}

func (s *InjectorTestSuite) TestSharedDependencyInitialisedCorrectly() {
	type Child struct {
		Int int
	}

	type Obj struct {
		Child Child
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: Obj{}},
			{Name: "github.com/j7mbo/goij/test.Child", Implementation: Child{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(Child{Int: 42})

	s.Assert().Equal(&Obj{Child{Int: 42}}, ij.Make("github.com/j7mbo/goij/test.Obj"))
}

func (s *InjectorTestSuite) TestSharedPointerDependencyInitialisedCorrectly() {
	type Child struct {
		Int int
	}

	type Obj struct {
		Child *Child
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: Obj{}},
			{Name: "github.com/j7mbo/goij/test.Child", Implementation: Child{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(Child{Int: 42})

	s.Assert().Equal(&Obj{&Child{Int: 42}}, ij.Make("github.com/j7mbo/goij/test.Obj"))
}

func (s *InjectorTestSuite) TestDependencyNotFoundInRegistryPanics() {
	type Dep struct{}
	dep := &Dep{}

	obj := struct {
		Dep *Dep
	}{
		Dep: dep,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: obj},
		},
	}

	s.Assert().Panics(func() {
		Goij.NewInjector(TypeRegistry.New(registry), nil).Make("github.com/j7mbo/goij/test.Obj")
	})
}

func (s *InjectorTestSuite) TestMakeEncounteringInterfaceDepWithNoConcretesPanics() {
	type Interface interface {
		AMethod()
	}
	type Obj struct {
		Dep Interface
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.Obj", Implementation: Obj{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.Interface", Implementation: (*Interface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	/* Note how Dep has no concrete. */
	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.Obj")
	})
}

func (s *InjectorTestSuite) TestMakeEncounteringInterfaceWithMultipleConcretesPanics() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObj2", Implementation: testObj2{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.testInterface")
	})
}

func (s *InjectorTestSuite) TestMakeEncounteringInterfaceWithZeroConcretesPanics() {
	registry := TypeRegistry.Registry{
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.testInterface")
	})
}

func (s *InjectorTestSuite) TestMakeEncounteringInterfaceWithExactlyOneConcreteReturnsObject() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&testObj{}, ij.Make("github.com/j7mbo/goij/test.testInterface"))
}

func (s *InjectorTestSuite) TestMakeEncounteringDependencyOfKindInterfaceWithExactlyOneConcreteReturnsObject() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	expectedObj := &testObjToMake{Dep: &testObj{}}

	s.Assert().Equal(expectedObj, ij.Make("github.com/j7mbo/goij/test.testObjToMake"))
}

func (s *InjectorTestSuite) TestMakeEncounteringDependencyOfKindInterfaceWithZeroConcretesPanics() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.Interface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.testObjToMake")
	})
}

func (s *InjectorTestSuite) TestMakeEncounteringDependencyOfKindInterfaceWithMultipleConcretesPanics() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObj2", Implementation: testObj2{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.testObjToMake")
	})
}

func (s *InjectorTestSuite) TestSharedObjectIsProvisionedWhenMakeEncountersDependencyOfKindInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Share(testObj{})

	ij.Make("github.com/j7mbo/goij/test.testObjToMake")
}

func (s *InjectorTestSuite) TestSharedObjectOfKindPointerIsProvisionedWhenMakeEncountersDependencyOfKindInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Share(testObj{})

	ij.Make("github.com/j7mbo/goij/test.testObjToMake")
}

func (s *InjectorTestSuite) TestCanBindInterfaceToConcreteWhenTheyExistInRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Bind("github.com/j7mbo/goij/test.testInterface", "github.com/j7mbo/goij/test.testObj")
}

func (s *InjectorTestSuite) TestMakeUsesSharedObjectFromATypeBoundToAnInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObj2", Implementation: testObj2{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Bind("github.com/j7mbo/goij/test.testInterface", "github.com/j7mbo/goij/test.testObj2")
	ij.Share(testObj{})
	ij.Make("github.com/j7mbo/goij/test.testInterface")
}

func (s *InjectorTestSuite) TestMakeUsesSharedObjectFromATypeBoundToAnInterfaceForAFoundDependency() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.testObj2", Implementation: testObj2{}},
			{Name: "github.com/j7mbo/goij/test.testObjToMake", Implementation: testObjToMake{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Bind("github.com/j7mbo/goij/test.testInterface", "github.com/j7mbo/goij/test.testObj2")
	ij.Share(testObj{})
	ij.Make("github.com/j7mbo/goij/test.testObjToMake")
}

func (s *InjectorTestSuite) TestBindingInterfaceToConcreteWhenConcreteDoesNotExistInRegistryPanics() {
	registry := TypeRegistry.Registry{
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Bind("github.com/j7mbo/goij/test.testInterface", "github.com/j7mbo/goij/test.testObj")
	})
}

func (s *InjectorTestSuite) TestBindingInterfaceToConcreteWhenInterfaceDoesNotExistInRegistryPanics() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Bind("github.com/j7mbo/goij/test.testInterface", "github.com/j7mbo/goij/test.testObj")
	})
}

func (s *InjectorTestSuite) TestCanMakeStructWithDelegatedAnonymousFuncForFactory() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", func() testObjWithInt {
		return testObjWithInt{Int: 42}
	})

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructPointerWithDelegatedAnonymousFuncForFactory() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", func() *testObjWithInt {
		return &testObjWithInt{Int: 42}
	})

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructPointerWithDelegatedFunctionLiteralForFactory() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", FactoryForObjWithInt)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructAutomatedWithSingleFactoryInFactoryRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryForObjWithInt}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructAutomatedWithSingleFactoryInFactoryRegistryForInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementation: (*testInterfaceForObjWithInt)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", FactoryForObjWithInt)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testInterfaceForObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructDependencyWithSingleFactoryInFactoryRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory", Implementation: testObjWithDepCreatedByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryForObjWithInt}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory").(*testObjWithDepCreatedByFactory).TestObjWithInt.Int)
}

func (s *InjectorTestSuite) TestCanMakeInterfaceDependencyWithSingleFactoryInFactoryRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInterfaceDepCreateByFactory", Implementation: testObjWithInterfaceDepCreateByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementation: (*testInterfaceForObjWithInt)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", FactoryForObjWithInt)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInterfaceDepCreateByFactory").(*testObjWithInterfaceDepCreateByFactory).TestInterfaceForObjWithInt.(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanDefineScalarParameterOnDependency() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Define("github.com/j7mbo/goij/test.testObjWithInt", "Int", 42)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanDefineNonParameterOnDependency() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Define("github.com/j7mbo/goij/test.testObjWithInt", "Int", 42)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanMakeStructWithShortNameWhenOneExists() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&testObjWithInt{}, ij.Make("testObjWithInt").(*testObjWithInt))
}

func (s *InjectorTestSuite) TestCanMakeInterfaceWithShortNameWhenOneExists() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(&testObj{}, ij.Make("testObj").(*testObj))
}

func (s *InjectorTestSuite) TestInterfaceDependencyWithInterfaceNotExistingInRegistryPanics() {
	myObj := struct {
		TestObj testInterface
	}{}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
			{Name: "github.com/j7mbo/goij/test.myObj", Implementation: myObj},
		},
	}

	s.Assert().Panics(func() {
		Goij.NewInjector(TypeRegistry.New(registry), nil).Make("github.com/j7mbo/goij/test.myObj")
	})
}

func (s *InjectorTestSuite) TestDependencyWithMultipleFactoriesInRegistryCausesPanic() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory", Implementation: testObjWithDepCreatedByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryForObjWithInt, FactoryForObjWithInt}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Panics(func() {
		ij.Make("github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory")
	})
}

func (s *InjectorTestSuite) TestCanBindShortInterfaceNameToShortStructName() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: testObj{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Bind("testInterface", "testObj")

	s.Assert().Equal(&testObj{}, ij.Make("testInterface"))
}

func (s *InjectorTestSuite) TestCanDefineScalarWithStructShortName() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Define("testObjWithInt", "Int", 42)

	s.Assert().Equal(42, ij.Make("testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanDefineScalarsAndShareInstanceAsExpected() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Share(testObjWithInt{Int: 42})

	s.Assert().Equal(42, ij.Make("testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanDelegateWithShortName() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("testObjWithInt", func() *testObjWithInt {
		return &testObjWithInt{Int: 42}
	})

	s.Assert().Equal(42, ij.Make("testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanInvokePointerReceiverMethodOnObject() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(testObjWithInt{Int: 42})

	s.Assert().Equal(42, ij.Invoke(ij.Make("testObjWithInt"), "IntMethod")[0])
}

func (s *InjectorTestSuite) TestCanInvokeValueReceiverMethodOnObject() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(42, ij.Invoke(ij.Make("testObjWithInt"), "ValueReceiverMethod")[0])
}

func (s *InjectorTestSuite) TestAutomaticFactoryIsRepeatedlyUsedAndResultIsNotCachedForMake() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithRandomInt}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotEqual(ij.Make("testObjWithInt"), ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestDelegateIsRepeatedlyUsedAndResultIsNotCachedForMake() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("testObjWithInt", FactoryWithRandomInt)

	s.Assert().NotEqual(ij.Make("testObjWithInt"), ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestAutomaticFactoryIsRepeatedlyUsedAndResultIsNotCachedForDependency() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory", Implementation: testObjWithDepCreatedByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithRandomInt}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	/* If there's a bug, this would cache the factory. */
	ij.Make("testObjWithDepCreatedByFactory")

	s.Assert().NotEqual(ij.Make("testObjWithInt"), ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestDelegateIsRepeatedlyUsedAndResultIsNotCachedForDependency() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory", Implementation: testObjWithDepCreatedByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("testObjWithInt", FactoryWithRandomInt)

	/* If there's a bug, this would cache the factory. */
	ij.Make("testObjWithDepCreatedByFactory")

	s.Assert().NotEqual(ij.Make("testObjWithInt"), ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestUserDelegateArgsAreResolved() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testDepForFactoryWithArgs", Implementation: testDepForFactoryWithArgs{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(testDepForFactoryWithArgs{Int: 42})
	ij.Delegate("testObjWithInt", FactoryWithArgs)

	s.Assert().Equal(42, ij.Make("testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestAutoFactoryArgsAreResolved() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testDepForFactoryWithArgs", Implementation: testDepForFactoryWithArgs{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithArgs}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(testDepForFactoryWithArgs{Int: 42})

	s.Assert().Equal(42, ij.Make("testObjWithInt").(*testObjWithInt).Int)
}

/*
This test demonstrates a recursive problem with delegates:

Make an object with an auto factory, factory requires interface argument, interface argument auto-resolved to concrete
type, concrete type has a factory registered for it. Logic added in code is, if delegate returns a type that has the
same delegate, don't call it. This is getting crazy...
*/
func (s *InjectorTestSuite) TestAutoFactoryArgsAreResolvedForAutoInterfaces() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceReturningInt", Implementation: (*testInterfaceReturningInt)(nil)},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithInterfaceArg}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().Equal(testObjWithInt{Int: 42}, ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestUserDelegateArgsAreResolvedForAutoInterfaces() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceReturningInt", Implementation: (*testInterfaceReturningInt)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("testObjWithInt", FactoryWithInterfaceArg)

	s.Assert().Equal(testObjWithInt{Int: 42}, ij.Make("testObjWithInt"))
}

func (s *InjectorTestSuite) TestLambdaInvocationWorksWithMake() {
	blah := struct {
		Int int
	}{
		Int: 42,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(blah)
	ij.Delegate("testObjWithInt", func(intGetter struct{ Int int }) *testObjWithInt {
		return &testObjWithInt{Int: intGetter.Int}
	})

	s.Assert().Equal(&testObjWithInt{Int: 42}, ij.Make("github.com/j7mbo/goij/test.testObjWithInt"))
}

func (s *InjectorTestSuite) TestGlobalDefinitionWorksForDependency() {
	blah := struct{ SomeVar int }{}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.blah", Implementation: blah},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.DefineGlobal("SomeVar", 42)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.blah").(*struct{ SomeVar int }).SomeVar)
}

func (s *InjectorTestSuite) TestGlobalDefinitionDoesNotOverrideObjectDefinition() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Define("github.com/j7mbo/goij/test.testObjWithInt", "Int", 42)
	ij.DefineGlobal("Int", 69)

	s.Assert().Equal(42, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(*testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestScalarOnLambdaDefinitionBecomesZeroValue() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	ij.Delegate("github.com/j7mbo/goij/test.testObjWithInt", func(SomeVar int) testObjWithInt {
		return testObjWithInt{Int: SomeVar}
	})

	s.Assert().Equal(0, ij.Make("github.com/j7mbo/goij/test.testObjWithInt").(testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestScalarWithUserRegisteredFactoryBecomesZeroValue() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testDepForFactoryWithArgs", Implementation: testDepForFactoryWithArgs{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithIntArg}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.DefineGlobal("globallyDefineMePlease", 42)

	s.Assert().Equal(0, ij.Make("testObjWithInt").(testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestCanDelegateInterfaceToFactory() {
	/* For those times when there is no exported struct and you only have an interface to inject. */
	registry := TypeRegistry.Registry{
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterface", Implementation: (*testInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("testInterface", func() testInterface {
		return &testObj{}
	})

	s.IsType(&testObj{}, ij.Make("testInterface"))
}

func (s *InjectorTestSuite) TestCanDelegateForInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInjectorDep", Implementation: testObjWithInjectorDep{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij.Injector", Implementation: (*Goij.Injector)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("github.com/j7mbo/goij.Injector", func() Goij.Injector {
		return ij
	})

	s.Implements(new(Goij.Injector), ij.Make("testObjWithInjectorDep").(*testObjWithInjectorDep).Injector)
}

func (s *InjectorTestSuite) TestCanDefineStructForInjection() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithDepCreatedByFactory", Implementation: testObjWithDepCreatedByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Define("testObjWithDepCreatedByFactory", "TestObjWithInt", testObjWithInt{Int: 42})

	s.Equal(42, ij.Make("testObjWithDepCreatedByFactory").(*testObjWithDepCreatedByFactory).TestObjWithInt.Int)
}

func (s *InjectorTestSuite) TestAutomaticFactoryReturningInterfaceResolvesToCorrectBoundStruct() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInterfaceDepCreateByFactory", Implementation: testObjWithInterfaceDepCreateByFactory{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
			{Name: "github.com/j7mbo/goij/test.secondObjectImplementingIntMethod", Implementation: secondObjectImplementingIntMethod{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementation: (*testInterfaceForObjWithInt)(nil)},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementations: []interface{}{FactoryReturningInterface}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	/* If this works a test for a single non-bound struct would also pass. */
	ij.Bind("testInterfaceForObjWithInt", "testObjWithInt")

	/*
		Logic should be: encounter object, look at args, arg is interface, find factory for interface that returns interface
		then do usual logic to return single instance, bound instance or panic if multiple
	*/
	s.Equal(22, ij.Make("testObjWithInterfaceDepCreateByFactory").(*testObjWithInterfaceDepCreateByFactory).TestInterfaceForObjWithInt.IntMethod())
}

func (s *InjectorTestSuite) TestPointerToScalarIsResolvedToZeroValue() {
	pointerInt := new(int64)

	obj := struct {
		Int *int64
	}{
		Int: pointerInt,
	}

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObj", Implementation: obj},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Equal(pointerInt, ij.Make("github.com/j7mbo/goij/test.testObj").(*struct{ Int *int64 }).Int)
}

func (s *InjectorTestSuite) TestDependencyPointerToScalarIsResolvedToZeroValue() {
	pointerInt := new(int64)

	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithPointerInt", Implementation: testObjWithPointerInt{}},
			{Name: "github.com/j7mbo/goij/test.testParentObjForObjWithPointerInt", Implementation: testParentObjForObjWithPointerInt{}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.IsType(pointerInt, ij.Make("github.com/j7mbo/goij/test.testParentObjForObjWithPointerInt").(*testParentObjForObjWithPointerInt).Obj.Int)
}

func (s *InjectorTestSuite) TestScalarPointerDelegateArgSetToZeroValue() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementations: []interface{}{FactoryWithPointerScalarArg}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.IsType(0, ij.Make("testObjWithInt").(testObjWithInt).Int)
}

func (s *InjectorTestSuite) TestFactoryReturningInterfaceWorksWithInterfaceArgForObjRequiringInterface() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.objRequiringInterface", Implementation: objRequiringInterface{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
			{Name: "github.com/j7mbo/goij/test.objImplementingAnInterface", Implementation: objImplementingAnInterface{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.anInterface", Implementation: (*anInterface)(nil)},
			{Name: "github.com/j7mbo/goij/test.testInterfaceReturningInt", Implementation: (*testInterfaceReturningInt)(nil)},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.anInterface", Implementations: []interface{}{FactoryWithInterfaceArgReturningInterface}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotPanics(func() {
		_ = ij.Make("objRequiringInterface").(*objRequiringInterface)
	})
}

/* The problem was a delegate with an interface return type, but we didn't return a pointer to the struct. */
func (s *InjectorTestSuite) TestFactoryWithInterfaceArgAndReturningInterfaceCanInjectAndReturnAsExpected() {
	registry := TypeRegistry.Registry{}

	registry.RegistryStructs = append(registry.RegistryStructs, TypeRegistry.RegistryStruct{Name: "github.com/j7mbo/MethodCallRetrier.MethodCallRetrier", Implementation: MethodCallRetrier.MethodCallRetrier{}})
	registry.RegistryStructs = append(registry.RegistryStructs, TypeRegistry.RegistryStruct{Name: "github.com/j7mbo/goij/test.ObjRequiringRetrier", Implementation: ObjRequiringRetrier{}})
	registry.RegistryStructs = append(registry.RegistryStructs, TypeRegistry.RegistryStruct{Name: "github.com/j7mbo/MethodCallRetrier.MaxRetriesError", Implementation: MethodCallRetrier.MaxRetriesError{}})
	registry.RegistryInterfaces = append(registry.RegistryInterfaces, TypeRegistry.RegistryInterface{Name: "github.com/j7mbo/MethodCallRetrier.Retrier", Implementation: (*MethodCallRetrier.Retrier)(nil)})
	registry.RegistryFactories = append(registry.RegistryFactories, TypeRegistry.RegistryFactory{Name: "github.com/j7mbo/MethodCallRetrier.MethodCallRetrier", Implementations: []interface{}{MethodCallRetrier.New}})
	registry.RegistryFactories = append(registry.RegistryFactories, TypeRegistry.RegistryFactory{Name: "github.com/j7mbo/goij/test.ObjRequiringRetrier", Implementations: []interface{}{MCRFactory}})

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotPanics(func() {
		_ = ij.Make("github.com/j7mbo/goij/test.ObjRequiringRetrier").(*ObjRequiringRetrier)
	})
}

/* Error with this was within injecting args into factory. It's asking for logrus.Logegr, we have *logrus.Logger. */
func (s *InjectorTestSuite) TestFactoryWithPtrArgInjectsCorrectly() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.ObjWithLogrusDep", Implementation: ObjWithLogrusDep{}},
			{Name: "github.com/j7mbo/goij/test.ParentObjWithDep", Implementation: ParentObjWithDep{}},
			{Name: "github.com/sirupsen/logrus.Logger", Implementation: logrus.Logger{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.ObjWithLogrusDep", Implementations: []interface{}{FactoryWithPtrArg}},
			{Name: "github.com/sirupsen/logrus.Logger", Implementations: []interface{}{logrus.New}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotPanics(func() {
		_ = ij.Make("github.com/j7mbo/goij/test.ParentObjWithDep").(*ParentObjWithDep)
	})
}

/* Error with this was within injecting args into factory.  Call using *config.ElasticSearchConfiguration as type config.ElasticSearchConfiguration. */
func (s *InjectorTestSuite) TestFactoryWithSharedArgInjectsNonPointerArgCorrectly() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.ObjWithLogrusDep", Implementation: ObjWithLogrusDep{}},
			{Name: "github.com/j7mbo/goij/test.ParentObjWithDep", Implementation: ParentObjWithDep{}},
			{Name: "github.com/sirupsen/logrus.Logger", Implementation: logrus.Logger{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.ObjWithLogrusDep", Implementations: []interface{}{FactoryWithPtrArg}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(logrus.New())

	s.Assert().NotPanics(func() {
		_ = ij.Make("github.com/j7mbo/goij/test.ParentObjWithDep").(*ParentObjWithDep)
	})
}

/* Many times structs are private, with only an interface and a factory returning the interface. Perfectly valid. */
func (s *InjectorTestSuite) TestCanMakeInterfaceWithNoStructTypeInRegistry() {
	registry := TypeRegistry.Registry{
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementation: (*testInterfaceForObjWithInt)(nil)},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.testInterfaceForObjWithInt", Implementations: []interface{}{FactoryReturningInterface}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotPanics(func() {
		_ = ij.Make("github.com/j7mbo/goij/test.testInterfaceForObjWithInt").(*testObjWithInt)
	})
}

func (s *InjectorTestSuite) TestCanMakeObjWithInterfaceDependencyWithNoStructTypeInRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.objRequiringInterface", Implementation: objRequiringInterface{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.anInterface", Implementation: (*anInterface)(nil)},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.anInterface", Implementations: []interface{}{FactoryReturningAnInterface}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)

	s.Assert().NotPanics(func() {
		_ = ij.Make("github.com/j7mbo/goij/test.objRequiringInterface").(*objRequiringInterface)
	})
}

func (s *InjectorTestSuite) TestDelegateInterfaceDependencyWithNoStructTypeInRegistry() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.objRequiringInterface", Implementation: objRequiringInterface{}},
		},
		RegistryInterfaces: []TypeRegistry.RegistryInterface{
			{Name: "github.com/j7mbo/goij/test.anInterface", Implementation: (*anInterface)(nil)},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Delegate("anInterface", FactoryReturningAnInterface)

	s.Equal(42, ij.Make("github.com/j7mbo/goij/test.objRequiringInterface").(*objRequiringInterface).Obj.Y())

}

func (s *InjectorTestSuite) TestSharedDependencyUsedInDelegate() {
	registry := TypeRegistry.Registry{
		RegistryStructs: []TypeRegistry.RegistryStruct{
			{Name: "github.com/j7mbo/goij/test.ParentObjForObjWithSharedDep", Implementation: ParentObjForObjWithSharedDep{}},
			{Name: "github.com/j7mbo/goij/test.ObjWithSharedDep", Implementation: ObjWithSharedDep{}},
			{Name: "github.com/j7mbo/goij/test.testObjWithInt", Implementation: testObjWithInt{}},
		},
		RegistryFactories: []TypeRegistry.RegistryFactory{
			{Name: "github.com/j7mbo/goij/test.ObjWithSharedDep", Implementations: []interface{}{NewObjWithSharedDep}},
		},
	}

	ij := Goij.NewInjector(TypeRegistry.New(registry), nil)
	ij.Share(testObjWithInt{Int: 128})

	s.Equal(128, ij.Make("github.com/j7mbo/goij/test.ParentObjForObjWithSharedDep").(*ParentObjForObjWithSharedDep).ObjWithSharedDep.TestObjWithInt.Int)
}

/* Types must be declared here to be found, can't add methods to a type within a function. */
type testInterface interface{ AMethod() }
type testObj struct{}
type testObj2 struct{}
type testObjToMake struct{ Dep testInterface }
type testObjWithInt struct{ Int int }
type testObjWithDepCreatedByFactory struct{ TestObjWithInt testObjWithInt }
type testObjWithInterfaceDepCreateByFactory struct{ TestInterfaceForObjWithInt testInterfaceForObjWithInt }
type testObjWithInjectorDep struct{ Injector Goij.Injector }
type testObjWithPointerInt struct{ Int *int64 }
type testParentObjForObjWithPointerInt struct{ Obj testObjWithPointerInt }

func (*testObj) AMethod()                       {}
func (*testObj2) AMethod()                      {}
func (t *testObjWithInt) IntMethod() int        { return t.Int }
func (*testObjWithInt) ReturnInt() int          { return 42 }
func (testObjWithInt) ValueReceiverMethod() int { return 42 }

type testInterfaceForObjWithInt interface {
	IntMethod() int
}

type secondObjectImplementingIntMethod struct{}

func (*secondObjectImplementingIntMethod) IntMethod() int {
	return 69
}

func FactoryForObjWithInt() *testObjWithInt {
	return &testObjWithInt{Int: 42}
}

func FactoryWithRandomInt() *testObjWithInt {
	return &testObjWithInt{Int: rand.Int()}
}

type testDepForFactoryWithArgs struct {
	Int int
}

func FactoryWithArgs(t *testDepForFactoryWithArgs) *testObjWithInt {
	return &testObjWithInt{Int: t.Int}
}

type testInterfaceReturningInt interface {
	ReturnInt() int
}

func FactoryWithInterfaceArg(t testInterfaceReturningInt) testObjWithInt {
	return testObjWithInt{Int: t.ReturnInt()}
}

func FactoryWithIntArg(globallyDefineMePlease int) testObjWithInt {
	return testObjWithInt{Int: globallyDefineMePlease}
}

func FactoryReturningInterface() testInterfaceForObjWithInt {
	return &testObjWithInt{Int: 22}
}

func FactoryWithPointerScalarArg(int *int) testObjWithInt {
	return testObjWithInt{Int: *int}
}

// ----- For test: TestFactoryReturningInterfaceWorksWithInterfaceArgForObjRequiringInterface()
type objRequiringInterface struct {
	Obj anInterface
}

type anInterface interface {
	Y() int
}

type objImplementingAnInterface struct{}

func (obj *objImplementingAnInterface) Y() int { return 42 }

func FactoryWithInterfaceArgReturningInterface(t testInterfaceReturningInt) anInterface {
	return &objImplementingAnInterface{}
}

func FactoryReturningAnInterface() anInterface {
	return &objImplementingAnInterface{}
}

// ----- For test: TestFactoryWithInterfaceArgAndReturningInterfaceCanInjectAndReturnAsExpected()

func MCRFactory(testInt int64, retrier MethodCallRetrier.Retrier) *ObjRequiringRetrier {
	return &ObjRequiringRetrier{Retrier: retrier}
}

type ObjRequiringRetrier struct {
	Retrier MethodCallRetrier.Retrier
}

// ----- For test: TestFactoryWithPtrArgInjectsCorrectly()

type ParentObjWithDep struct {
	ObjWithLogrusDep ObjWithLogrusDep
}

type ObjWithLogrusDep struct {
	Logger logrus.Logger
}

func FactoryWithPtrArg(arg logrus.Logger) *ObjWithLogrusDep {
	return &ObjWithLogrusDep{Logger: arg}
}

// ----- For test: TestSharedDependencyUsedInDelegate()

type ObjWithSharedDep struct {
	TestObjWithInt *testObjWithInt
}

type ParentObjForObjWithSharedDep struct {
	ObjWithSharedDep ObjWithSharedDep
}

func NewObjWithSharedDep(TestObjWithInt *testObjWithInt) ObjWithSharedDep {
	return ObjWithSharedDep{TestObjWithInt: TestObjWithInt}
}
