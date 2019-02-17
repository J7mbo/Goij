package TypeRegistry

import (
	"fmt"
	"math/rand"
	"os"
	"text/template"
	"time"
)

/* Basically writing my own template because I haven't been bothered to learn the template package yet. */
var (
	fileTop     = "package TypeRegistry\n"
	fileImports = ""
	fileFuncDef = "\nfunc GetAutoGeneratedRegistry() map[string][]interface{} {\n"
	fileMiddle  = "    registry := make(map[string][]interface{})\n"
	fileBottom  = "\n    return registry\n}\n"
)

/* Default file contents for resetting. */
var (
	defaultFile = `package TypeRegistry

func GetAutoGeneratedRegistry() map[string][]interface{} {
	registry := make(map[string][]interface{})

    return registry
}
`
)

/* An object responsible for writing PackageData structs to a file given the template above. */
type AutoGeneratedRegistryWriter struct{}

/* Write the contents of defaultFile to the file. */
func (*AutoGeneratedRegistryWriter) WriteDefaultFile(filePath string) {
	file, err := os.Create(filePath)

	defer func() { _ = file.Close() }()

	if err != nil {
		panic(fmt.Sprintf("Unable to write empty type registry to file: %s, error: %s", filePath, err.Error()))
	}

	_, err = file.WriteString(defaultFile)

	if err != nil {
		panic(fmt.Sprintf("Unable to write empty type registry to file: %s, error: %s", filePath, err.Error()))
	}
}

/* Write the results of []PackageData to a readable to file. Could've used the AST for this, but I'm lazy... */
func (*AutoGeneratedRegistryWriter) WriteAutoGeneratedDataToFile(packageDataList []PackageData, filePath string) {
	for _, packageData := range packageDataList {
		/* Again, if we have a package but no files, don't bother writing the import. */
		if len(packageData.Structs) == 0 {
			continue
		}

		/* Alias for the import so multiple of the same package name will not be an issue. */
		alias := generateRandomString(8)

		fileImports += fmt.Sprintf("import %s \"%s\"\n", alias, packageData.ImportPath)

		for _, structName := range packageData.Structs {
			mapKey := fmt.Sprintf("%s.%s", packageData.PackageName, structName)
			mapType := fmt.Sprintf("%s.%s", alias, structName)

			/* Ensure that we don't duplicate keys in the registry. */
			//numSameTypes := strings.Count(fileMiddle, mapKey)

			/* Looks like: registry["Injector.TypeRegistry"] = new(Injector.TypeRegistry) */
			fileMiddle += fmt.Sprintf(
				"\n    registry[\"%s\"] = append(registry[\"%s\"], *new(%s))", mapKey, mapKey, mapType,
			)
		}
	}

	file, err := os.Create(filePath)

	defer func() { _ = file.Close() }()

	if err != nil {
		panic(fmt.Sprintf("Unable to write auto generated type registry to file: %s, error: %s", filePath, err.Error()))
	}

	fileContents := fileTop + fileImports + fileFuncDef + fileMiddle + fileBottom

	_ = template.Must(template.New("").Parse(fileContents)).Execute(file, struct{}{})
}

/* Generate a pseudorandom alphanumeric string of a fixed length. */
func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())

	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, length)

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}
