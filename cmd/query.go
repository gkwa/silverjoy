package cmd

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run a Neo4j query and generate an HTML report",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("query called")
		if err := runQueryAndGenerateHTML(); err != nil {
			slog.Error("runQueryAndGenerateHTML", "error", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	// Add flags and configurations if needed
}

func runQueryAndGenerateHTML() error {
	// Neo4j connection setup

	// Create a new context
	ctx := context.Background()

	dbUri := "neo4j://localhost"
	dbUser := ""
	dbPassword := ""
	driver, err := neo4j.NewDriverWithContext(
		dbUri,
		neo4j.BasicAuth(dbUser, dbPassword, ""),
	)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return err
	}

	cypherQuery := `
		MATCH (product:Product)
		WITH product.name AS productName, p.urls AS productUrls
		UNWIND productUrls AS url
		RETURN productName, url
		ORDER BY toLower(productName)
	`

	queryParams := map[string]interface{}{}

	resultTransformer := neo4j.EagerResultTransformer

	databaseIdentifier := "neo4j"

	result, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		cypherQuery,
		queryParams,
		resultTransformer,
		neo4j.ExecuteQueryWithDatabase(databaseIdentifier),
	)
	if err != nil {
		slog.Error("error", "error", err)
		return err
	}

	// Process Neo4j query result
	products := make(map[string][]string)

	fmt.Println(result)
	for _, record := range result.Records {
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		productName, found := record.Get("productName")
		if !found {
			// Return an error if "productName" is not found in the record
			return fmt.Errorf("error: Field 'productName' not found in record")
		}

		url, found := record.Get("url")
		if !found {
			// Return an error if "url" is not found in the record
			return fmt.Errorf("error: Field 'url' not found in record")
		}

		products[productName.(string)] = append(products[productName.(string)], url.(string))
	}

	// Sorting keys and values
	var productNames []string
	for key := range products {
		productNames = append(productNames, key)
	}
	sort.Strings(productNames)

	for _, name := range productNames {
		urls := products[name]
		sort.Strings(urls)
	}

	// Generating HTML
	tmpl := template.Must(template.New("output").Funcs(template.FuncMap{
		"startsWithHTTP": func(s string) bool {
			return strings.HasPrefix(s, "http")
		},
	}).Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Output</title>
</head>
<body>
    <h1>Data:</h1>
    <ul>
        {{range $key := .Keys}}
        <li><strong>{{$key}}</strong>
            <ul>
                {{range $value := index $.Data $key}}
                <li>{{if startsWithHTTP $value}}<a href="{{$value}}" target="_blank">{{$value}}</a>{{else}}{{$value}}{{end}}</li>
                {{end}}
            </ul>
        </li>
        {{end}}
    </ul>
</body>
</html>
`))

	// Writing to HTML file
	file, err := os.Create("output.html")
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, map[string]interface{}{
		"Keys": productNames,
		"Data": products,
	})
	if err != nil {
		return err
	}

	return nil
}
