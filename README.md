![Goij - Go dependency injector](https://user-images.githubusercontent.com/2657310/53684520-b2327b00-3d0e-11e9-8f4b-d2e00a30fcf7.png)

[![Build Status](https://travis-ci.com/J7mbo/Goij.svg?token=yHmxZpU2vJZUs1GXsdCa&branch=master)](https://travis-ci.com/J7mbo/Goij)
[![codecov](https://img.shields.io/codecov/c/token/FND4rf6uVh/github/j7mbo/goij.svg)](https://codecov.io/gh/J7mbo/Goij)
[![GoDoc](https://godoc.org/github.com/J7mbo/Goij?status.svg)](https://godoc.org/github.com/J7mbo/Goij)
[![Release: Alpha](https://img.shields.io/badge/release-alpha-blue.svg)](https://github.com/j7mbo/Goij)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE.md)

Goij is a recursive dependency injector. Use Goij to bootstrap and wire together S.O.L.I.D, clean Go applications.

### Example

```Go
type X struct { Dependency Y }
type Y struct { Dependency Z }
type Z struct { }

x := injector.Make("X").(*X)
z := x.Dependency.Dependency // z is == Z{}.
```

### Features

Some of the features of the injector are as follows:

- Recursive initialisation of all public fields on structs
- Create structs from strings
- Binding of interfaces to a specific implementation
- Shared objects to inject copies of the same instance around the application
- Delegation of struct initialisation to factories
- A very simple API

### How it works

Among other things, Goij instantiates public struct dependencies based on their types at runtime. This requires the use
of reflection and a type registry, which can be auto-generated. This allows structs to be initialised from strings.

### Actual real usage

You can see how I'm using the injector in a real live microservice using GRPC, CQRS, some DDD and other goodies 
[here](https://github.com/J7mbo/palmago-streetview).

Quick Api Overview
-

##### `Make(structName string)`

`Make()` initialises objects, and their public fields, recursively. If any of these objects have a factory, have been
`Share()`d previously, or have a user-provided `Delegate()` (factory) with the `New` convention, these are invoked to 
return the relevant field.

The `objectName` parameter can be either the fully qualified name of the struct, which must exist in the `TypeRegistry`,
or it can be the short name. For example: `my/app/Logger.Logger` or `Logger.Logger`. Short names only work when there
is only one `Logger.Logger` in the registry. It is advised to use the fully qualified name to avoid issues.

Read more in [Basic Recursive Instantiation](#basic-recursive-instantiation).

##### `Bind(interfaceName string, structName string)` 

`Bind()` tells the injector that, upon encountering an interface for one of the fields during recursive initialisation,
to inject the given specific struct type implementing that interface.

This also works when `Delegate()` arguments are encountered of the interface kind.

Note that if there is exactly one implementing interface in the type registry then you do not need to `Bind()` that
single specific implementation as it will be automatically provisioned and injected for you.

Read more in [Interface Binding](#interface-binding).

##### `Share(object interface{})`

`Share()` tells the injector that, upon encountering a type that matches the type of the `Share()`d object during
recursive initialisation, to inject this specific already initialised object instead of provisioning a new one. Any
properties of this injected `Share()`d instance are not recursively provisioned.

##### `Delegate(structName string, factoryMethod interface{})`

`Delegate()` tells the injector that, upon encountering the given struct during recursive initialisation, to invoke
the given factory function that returns this object. The factory function can also be a lambda.

Note that if there is exactly one factory in the type registry then you do not need to `Delegate()` that single specific
factory as it will be automatically invoked and the result injected for you.

Read more in [Initialisation Delegates](#initialisation-delegates).

There's a lot more that the injector can do for us, so let's move onto the guide.

The Guide
-

#### The Injector

* [Creator Note](#creator-note)
* [Requirements and Installation](#requirements-and-installation)
* [Initialisation](#initialisation)
* [Basic Recursive Instantiation](#basic-recursive-instantiation)
* [Interface Binding](#interface-binding)
* [Injection Definitions](#injection-definitions)
* [Instance Sharing](#instance-sharing)
* [Initialisation Delegates](#initialisation-delegates)
* [Injecting third-party Dependencies](#third-party-dependencies)
* [Example Use-Cases](#example-use-cases)
* [FAQ](#FAQ)

#### The Type Registry

* [Generating](#generating)
* [Build your own](#building-your-own)

### Creator note

Contributors are welcome! There are many things to be improved in Goij before it is released out of alpha. Here are some
of the potential improvements. Your input, thoughts and PRs are very welcome, so open an issue and let's talk. I'd
also appreciate the help in getting this to something much more usable.

- Better visualisation and logging for the object initialisation path, more standardised logging message to help users
debug their problems much faster than currently - maybe even a UI for this as debugging is a nightmare right now
- Documentation in the form of a diagram on the logic and ordering of injection, depending on delegates etc
- Make logger optional through variadic `InjectionConfiguration` arguments to simplify `New()`
- Add more logging in all the places it is necessary (factories, for example)
- Document logger and `InjectionConfiguration` options
- Change from panics to errors? A discussion on this is needed
- Do not inject a copy on encountering a pointer; allow the user to utilise the same address, blame them when things go wrong

## Requirements and Installation

###### Requirements

- Goij requires you to be using go modules

###### Installation

    go get github.com/j7mbo/goij

## Setup

Because Go doesn't actually allow you to create an object from a string due to the lack of a global registry, you need a
central registry of types. You can either build one yourself, or you can generate one automatically with the gen binary
included with the injector.

The location of this binary will depend on how you get it. If you do a `go get`, you'll have to find it in your go
directory and execute it there.

```bash
../path_to_goij/bin/gen -o Registry.go -dir ./src/ -exclude oneDir -exclude twoDir
```
    
This generates an output file (`-o`) containing a registry of all public structs, interfaces and factory functions 
within the given directory (`-dir`). You can then feed these maps to the injector for later initialisation.

You can also compile this yourself. You can google the correct arguments for your arch. The included one is built on
MacOS.

```bash
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ../path_to_goij/bin/gen ./gen
```

## Basic Usage

###### Initialisation

To start using the injector, simply create a type registry, and pass it into the `Injector` factory method.

```go
    registry := GetRegistry() // After running ../path_to_vendor/bin/gen

    injector := Goij.NewInjector(TypeRegistry.New(registry), nil)
```

###### Basic Recursive Instantiation

If a struct only asks for struct dependencies, you can use the injector to initialise and inject them without specifying 
any injection definitions.

```go
type Object struct{
    Dep DepOne
}

type DepOne struct{}

object := injector.Make("Object").(*Object) // Compiles and runs successfully.
```

This is fully recursive, so any dependencies of `DepOne` would also be initialised and injected, and so on and so forth.

###### Interface Binding

You may have noticed that the previous example demonstrated initialisation of structs with struct dependencies.
Obviously, many of your objects won't fit this mould. Some structs will depend on interfaces.

In the case that there is exactly one struct implementing an interface then that one implementing struct will be 
provisioned and injected automatically.

```go
type Object struct{
    InterfaceDependency AnInterface
}

type AnInterface interface {
    AMethod()
}

type DepOne struct{}
func (*DepOne) AMethod() {}

dep := injector.Make("Object").(*Object).InterfaceDependency // Instance of DepOne.  
```

You can also create an interface directly using the same logic:

```go
dep := injector.Make("AnInterface") // Instance of DepOne.
```

In the case that there are more than one structs implementing an interface in the type registry, we will need to assist 
the injector by telling it exactly which struct to inject when the interface is encountered during recursive 
initialisation.

```go
type Object struct{
    InterfaceDependency AnInterface
}

type AnInterface interface {
    AMethod()
}

type DepOne struct{}
func (*DepOne) AMethod() {}

type DepTwo struct{}
func (*DepTwo) AMethod() {}

injector.Bind("AnInterface", "DepTwo")
injector.Make("Object").(*Object).InterfaceDependency // Instance of DepTwo.
```

###### Injection Definitions

In the case that you want a specific instance of an object injected in a type when the injector encounters it, you can
do this with `Define()`:

 ```go
type Dependency struct {
    Int int
} 

type Object struct {
    Dep Dependency
}

injector.Define("Object", "Dep", Dependency{Int: 42})

injector.Make("Object").(*Object).Dependency.Int // 42.
```

Note that the above overrides `Share()`, so `Object` will always get a `Dependency` with `Int` being 42, but all the
rest of the objects that depend on `Dependency` will have either the `Share()`d object injected, or else an empty object
with `Int` being set to it's zero value. 

Some of your structs will specify primitive types such as strings and integers. In such cases we need to assist the 
Injector by telling it exactly what we want to inject.

```go
type Object struct {
    HostName string
    Port int
}

injector.Define("Object", "HostName", "http://www.github.com")
injector.Define("Object", "Port", 80)

injector.Make("Object").(*Object).Port // 80.
```

###### Globally defined injection definitions for scalar types

If you know that you want to always inject a value for a given field name, you can do do this as follows:

```go
type Object struct {
    AFieldHere string
}

injector.DefineGlobal("AFieldHere", "Hello World")

injector.Make("Object").(*Object).AFieldHere // Hello World.
```

> ***Note***: *Globally defined definitions should be used with care as the matching is only done on parameter name.*

###### Instance Sharing

One of the problems plaguing software architecture in Go is utilising global state to pass around objects. In fact, Go's
context package was built for the purpose of sharing cancellable channels around the application simply because they are
not being dependency injected. Goij makes this problem a triviality with the ability to inject the same value around the
application.

```go
type Database struct {
    Config DBConfiguration
}

injector.Share(DBConfiguration{ Hostname: "http://www.github.com", Port: 80 })

injector.Make("Database").(*Database).Config.Hostname // http://www.github.com.
```

Instance sharing enables the single-time initialisation of, for example, a `Configuration` object, a `Logger`, a 
`Database` connection or even the sharing of the `Context` from the context package in your composition root without 
injecting factories where they are not needed and without duplicating initialisation code everywhere.

###### Initialisation Delegates

Often the factory method pattern is used to initialise an object. Goij allows you to add factories into the injection 
process by specifying initialisation delegates on a per-class basis. In the case that there is single factory in the 
type registry for the requested type, the factory function will automatically be used to initialise the object.  

Let's look at a very basic example to demonstrate the concept of initialisation delegates:

```go
type MyComplexService {
    SomeValue int
}

func SomehowBuildComplexService() *MyComplexService {
    val := calculateSomeValue() // Do some stuff, get 1337.
	
    return &MyComplexService{SomeValue: val}
}

type Controller {
    Service MyComplexService
}

injector.Delegate("MyComplexService", SomehowBuildComplexService)

injector.Make("Controller").(*Controller).Service.SomeValue // 1337.
```

If there are multiple factories in the type registry, or you want to explicitly tell the inject to use a factory not in 
the type registry, you can with a a first class function type as shown above, or with a lambda:

```go
injector.Delegate("MyComplexService", func() *MyComplexService {
    val := calculateSomeValue() // Do some stuff, get 1337.

    return &MyComplexService{SomeValue: val}
})
```

Initialisation delegates can also require parameters to fulfil their objectives as a factory. If your factory method
asks for dependencies, they will be provisioned and injected into the function upon invocation. 

```go
initialisedService := MyComplexService{}

injector.Share(initialisedService)

injector.Delegate("AnObject", func(s *MyComplexService) *AnObject {
    value := s.CalculateSomeValue()
    
    return &AnObject(val: Value)
})
```

There are also cases where a package only provides two things for the user: an interface and a factory that returns that
interface. Some consider this a best practice whilst others cry foul that it is not idiomatic. Regardless, the injector
can work with interface delegates also:

```go
type AnObject interface {}

injector.Delegate("AnObject", func() *AnObject {
    return AnObject.New()
})
``` 

> ***Note***: *Delegate dependency resolution works with structs and interfaces (resolved to the correct struct), but
not with scalar definitions (global or otherwise) because Go does not allow retrieving function argument names.*

###### Third-party Dependencies

To be able to inject third-party dependencies with the injector, they also need to be in the registry. You can generate
a separate "vendor registry" and pass it to `TypeRegistry.New()`. Simply change the function name in the generated 
registry file if you generate it into the same directory as the other registry.

*You may find it useful to first run `go mod vendor` to have a directory of dependencies just for your project.*

Generate a vendor registry from the vendor directory:

```bash
vendor/github.com/j7mbo/goij/bin/gen -o ./VendorRegistry.go -dir ./vendor/
```

Then rename the generated function to `GetVendorRegistry()`, and use it in Go like this:

```go
Goij.NewInjector(TypeRegistry.New(GetRegistry(), GetVendorRegistry()))
```

The injector will then be able to utilise your own registry and the vendor registry as well.

###### Invocation

The injector provides a handy method to call a method dynamically:

```go
injector.Invoke(TheObject{}, "methodName", arg1, arg2, etc)
```

## Dependency Resolution

Goij resolves dependencies in the following order:

- Factory / delegate, with arguments recursively initialised following the same logic
- Cached / shared object, no recursive initialisation here
- Recursive initialisation on object
- Defined scalars are injected, else the encounted scalar will be zero'd 

# The Type Registry

Go is a statically typed language, and there is no central registry of types available to the user. As a result, types
cannot be initialised from a string such as `var := "MyStruct"; obj := new(var)`. The type literal must exist in the
code for the Go compiler to compile and use it. As a result, a type registry is needed.

###### Generating

Goij includes a utility to generate a registry. It utilises the AST (abstract syntax tree) to search for all public
structs, interfaces and factory methods (more on the logic behind this below), and writes them to a file with their
fully qualified package name.

The reason fully qualified package names such as `github.com/j7mbo/goij/Injector.Injector` are used is because simply
using `injector.Make("Injector")` could not work if there are two types in the type registry with the same name,
ignoring the fact that two identical keys cannot exist in a `map[string]interface{}`.

The gen binary also tries to guess the package name for the generated file, but this is not always correct especially
when generating into the project's root directory. Be aware you may need to rename the package before using the file.

> ***Note***: *Whilst using short names is currently enabled for the public api, it is safer to rely on fully qualified 
package names when using Goij.* 

The gen command has the following options:

```
Usage:

   gen [arguments]
   
The arguments are:

     o        The output file such as "Registry.go" or "/path/to/dir/FileRegistry"
     dir      The directory to scan for structs, interfaces, factories etc
     exclude  A directory to exclude from searching (useful for vendor/ etc), can use multiple times in command
     reset    Resets the registry back to the default empty template if used with -o

All paths can be relative or absolute.
```

The logic for which types are added to the registry are as follows:

- All exported struct types are added
- All exported interface types that have at least one implementing exported struct are added
- All exported functions beginning with `New` (idiomatic convention) with a return type are added as factories
- These are written to a file containing the function: `func GetRegistry() Registry`, which you can feed to the injector
on initialisation.

###### Building your own

The types returned from registry generation are also available for the end user. Sometimes it is easier or preferable
to build your own registry.

The registry contains arrays of `TypeRegistry.RegistryStruct`, `TypeRegistry.RegistryInterface` and
`TypeRegistry.RegistryFactory`, typically looking like this:

```go
registry := TypeRegistry.Registry{
    RegistryStructs: []TypeRegistry.RegistryStruct{
        TypeRegistry.RegistryStruct{ Name: "github.com/j7mbo/goij/src/TypeRegistry.Registry", Implementation: TypeRegistry.Registry{}},
    },
    RegistryInterfaces: []TypeRegistry.RegistryInterface{
    	TypeRegistry.RegistryInterface{ Name: "github.com/j7mbo/Goij.Injector", Implementation: (*Injector)(nil)},
    },
    RegistryFactories: []TypeRegistry.RegistryFactory{
        TypeRegistry.RegistryFactory{ Name: "github.com/j7mbo/goij/src/TypeRegistry.AutoRegistryGenerator", Implementations: []interface{}{ TypeRegistry.NewAutoRegistryGenerator }})
    },
}
```

The `registry` variable is then ready to be passed to the injector and is used as a lookup for all string-related
operations during recursive initialisation.

Not only can you pass multiple registries to the injector as the `TypeRegistry.New` function signature accepts variadic 
`Registry` objects, but sometimes it may be preferable to do this to maintain separation between a registry of the types
in your own application and a registry of the types from a third party library for quick removal.

> ***Note***: *It is important that you follow the conventions of the registry above. You MUST NOT add pointers to the struct 
registry, interfaces must be nil interfaces to retrieve the type at runtime, factories MUST follow similar 
conventions, and names in the registry MUST be the fully qualified package path followed by the type name.*

## Example use-cases

Injectors should be used to wire together the disparate objects of your application into a cohesive functional unit,
generally at the bootstrap or front-controller stage of the application, also known as the 'composition root'. In Java 
and go this would be `main()`, in PHP it would be `index.php`, in swift it would be the `AppDelegate`.

> ***Note***: *Goij is NOT a service locator. DO NOT turn it into one. Service locator is an anti-pattern for rare and
very extreme edge-cases; it hides class dependencies, makes code more difficult to maintain, reason with and test, and
makes a liar of your object API! The only places that an injector should be used are the composition root or factories.*

###### Dynamic Routing

One such usage provides an elegant solution for one of the thorny problems in web applications: how to initialise and
utilise a route through the application dynamically where the needed dependencies are not known at compile-time and 
depend on the route hit by the user.

Here is an incomplete example of what can be achieved with the injector.

```yaml
routes:
    route: /
    controller: IndexController
    action: Invoke
```

```go
package main

func main() {
    injector := Goij.NewInjector(TypeRegistry.New(GetRegistry), nil)
    router   := mux.NewRouter()
    routes   := RouteLoader.Load("routes.yml")
	
    for _, routeData := range routes {
        router.HandleFunc(routes.route, func(w http.ResponseWriter, r *http.Request) {
            injector.Invoke(injector.Make(routes.controller), routes.action, w, r)
        })
    }
}
```

New routes can be added and used immediately which can help in rapid application development and this is just the start.

###### Correlation Id

In distributed systems, a correlation id is necessary to track messages between services and also throughout an
application. One practice is to pass the correlation id throughout the application, including into factories, and into
many places where the id is not even needed and is only required to be passed through to a sub-dependency.

With Goij, a dependency somewhere down the object hierarchy can simply add a public correlation id property and the 
shared id will be injected and ready to use, removing the need for ugly and unecessary APIs.  

It's entirely possible the Go application will be the first point where a correlation id is created, but imagine the
correlation id is retrieved from the request each time instead.

```go
package main

func main() {
    injector := Goij.NewInjector(TypeRegistry.New(GetRegistry), nil)
    
    injector.Delegate("CorrelationId", func(r *http.Request) {
        return r.Header.Get("correlation-id") // Eg: 1337.
    })
    
    handleRoutes(injector)
}

func handleRoutes(i Injector) {
    router   := mux.NewRouter()
    routes   := RouteLoader.Load("routes.yml")
    
    for _, routeData := range routes {
        router.HandleFunc(routes.route, func(w http.ResponseWriter, r *http.Request) {
            i.Share(r) // Important!

            i.Invoke(i.Make(routes.controller), routes.action, w, r)
        })
    }
}

type IndexController struct {
    MyService *Service
}

func (c *IndexController) Invoke() {
    c.MyService.DoSomething()
}

type MyService struct {
    CorrId CorrelationId
    Logger Logger
}

func (m *MyService) DoSomething() {
    m.Logger.Log("Something happened!", {correlationId: m.CorrID})
}
```

This is a contrived example. What you could do is inject the correlation id directly into the logger so that wherever
anything is logged there is always a correlation id available. However this example does show the ability to delegate
the creation of a correlation id to a factory retrieving the correlation id from a request header, and then using this
id anywhere it is required throughout the application.

###### An 'MVC' Framework in Go

Utilising the above example of dynamic routing, you can require model objects from the model layer by adding them as
public properties of your controllers, and require database access or more by adding them as public properties of your
model objects.

```go
package main

func main() {
    db := NewDb(
        os.Getenv("hostname"),
        os.GetPort("port"),
        /* etc etc... */
    )
    
    ij = Goij.NewInjector(TypeRegistry.New(GetRegistry()), nil)
    ij.Share(db)
    
    /* -- SNIP -- Perform routing and pass in request to Controller. -- SNIP -- */
    
    ij.Invoke(ij.Make("IndexController"), "Invoke", request, responseWriter)
}

/* Controller. */

type IndexController {
    Users Users
}

func (c *IndexController) Invoke(request http.Request, responseWriter http.ResponseWriter) {
    /* Naively assumes the user id is in a GET parameter for simplicity of example... */
    userId := request.URL.Query().Get("user_id")
    
    user := c.Users.FindById(userId)
    
    fmt.Fprintf(w, "Hello, your username is: %s", user.GetUsername())
}

/* User entity. */

type User struct {
    id string
    username string
}

func (u *User) GetUsername() string {
    return u.username
}

func (u *User) NewFromResultSet(resultSet []string) *User {
    return &User{id: resultSet[0]["id"], username: resultSet[0]["id"]}
}

/* User repository interface for the domain. */

type Users interface {
    FindById(id int) User
}

/* User repository implementation for the infrastructure layer. */

type UserRepository {
    /* Configured and share()d in the composition root (main). */
    DB Database
}

func (u *UserRepository) FindById(id int) *User {
    /* Obviously escape the id. */
    resultSet := u.DB.Query("SELECT * FROM users WHERE id = " + id)
    
    return                                                                                                                                                                                                                                                                  User.NewFromResultSet(resultSet)
}
```

As soon as the injector creates the `IndexController`, it also injects `Users`, with the concrete implementation being 
the `UserRepository`, which already has the `Database` provisioned and injected in.

In this example, assuming you had also shared a `Logger` in the composition root, you could add a `Logger` public
property in the controller, in the model, in the repository... anywhere, and immediately have it provisioned and ready
for use.

## Future ideas

- Provide an optional `InjectionConfiguration` to allow users to customise the injector:
    - Inject all public properties
    - Inject all properties with a given struct tag
    - Inject all private properties with a given struct tag
    - Only inject via factories ("constructor style")

## FAQ

> What is the current status of the project?

**Absolute massive here be dragons.** Goij is in active development and is **ALPHA**. This means you can expect the API 
to change before hitting v1.0.0 and, although it *is* being used on production in an enterprise environment, it is not 
currently considered stable. There are plenty of bugs/

> I got an error, why is the injector not working?!

Double check you are doing exactly what the documentation specifies before opening a new issue. Perhaps the
functionality you are looking for has not been implemented yet. The injector is currently in alpha mostly because it is
following 'the happy path' and doesn't provide particularly nice errors for things that could go wrong.

> What does the injector return, a pointer or a struct?

By default the injector returns pointers unless registered factories or delegates return otherwise. Either way, the
return values are copies of the originals, so that any changes are not propagated back to the type registry or cache and
therefore also not to any other places that they have been injected.

> What happens with pointers?

Goij injects *copies* of the type in the registry or the cache. If a dependency is of the pointer kind, Goij will inject
a pointer copy. Goij does not currently allow multiple pointers to be pointing to the same address in the registry or
cache as this would greatly increase complexity due to shared application state, race conditions, mutex usage etc.

In the future this might change. For example, the idea of duplicating a database connection isn't really necessary and
it would be better to actually inject the single instance of it, although this would be a pointer so could cause
problems if used over multiple threads. I'm open to discussing this in an issue.

> Is Goij thread safe?

Goij currently makes no promises on thread safety - you would possibly need to guard for this in your application.  

> Isn't copying objects everywhere expensive?

It is arguable that, whilst pointers can avoid copying memory, there are tradeoffs such as additional indirection and
increased work for the garbage collector. Computers are very fast at copying memory so do not just use pointers because
you think they might give you better performance. Default to using values except when you need the semantics a pointer
provides.

> Why does the type registry not use lambdas?

Potentially initialising empty structs for every type in the application on `GetRegistry()` can be costly. However if
you look at the memory consumption of an individual struct with several properties, the memory allocation is actually
very small.

The problem with utilising lambdas would be that every one of these would need to be executed (they would return 
`interface{}`) and then reflected on to get the return type when searching the type registry, and this could be costly.

Given a struct containing the following properties: 3 struct pointers, 2 strings, 2 bools and an int32, the size in
memory in bytes is ~40 bytes. [See here](https://play.golang.org/p/NMKKWMviETN). Assuming a type registry containing
1000 structs contains the aforementioned properties, the total size in memory would be ~40kb. The memory overhead is
more acceptable than the additional computation required with lambdas.

> Reflection is slow

You may have heard that "reflection is slow". Let's clear something up: anything can be "slow" if you're doing it wrong.
Reflection is an order of magnitude faster than disk access and several orders of magnitude faster than retrieving 
information (for example) from a remote database. Go, as a language, is extremely fast in it's own right. Goij caches 
some of the structs it encounters to minimize the potential performance impact. After the initial caching, 
injection is as fast as a map lookup.

> Go was not designed for this

Go provides not only a powerful language but also a toolset enabling developers to build anything they find useful.

> I don't like magic

That depends on how you define magic. Magic at a distance is bad. A well-tested isolated box can be magic to some, or
useful to others.

> Dependency Injection Container XYZ already exists

Goij is not a dependency injection container, and it has a very simple API compared to many other libraries.

> I found a bug or a better way of implementing something

Please read [CONTRIBUTING.md](CONTRIBUTING.md) and feel free to open a
[new Github issue](https://github.com/J7mbo/Goij/issues/new) where we can discuss it.

> How can I recompile the gen command?

This depends on your environment, so check out the docs and look for `GOOS` and `GOARCH` flags. Here's what Goij used:

```bash
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o bin/gen ./cmd/gen/
```
