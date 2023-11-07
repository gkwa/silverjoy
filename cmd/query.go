package cmd

import (
	"context"
	"fmt"
	"html/template"
	"os"

	"github.com/spf13/cobra"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("query called")
		test()
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	// Here you will define your flags and queryuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// queryCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// queryCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type RecipeIngredient struct {
	Ingredients []string
	Stores      []string
}

func test() {
	ctx := context.Background()
	// URI examples: "neo4j://localhost", "neo4j+s://xxx.databases.neo4j.io"
	dbUri := "neo4j://localhost"
	dbUser := ""
	dbPassword := ""
	driver, err := neo4j.NewDriverWithContext(
		dbUri,
		neo4j.BasicAuth(dbUser, dbPassword, ""))
	defer driver.Close(ctx)
	if err != nil {
		panic(err)
	}

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		panic(err)
	}

	// Create a session with write access

	// Create a session with write access
	session := driver.NewSession(
		ctx,
		neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		},
	)

	defer session.Close(ctx)

	// Begin an explicit transaction
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		fmt.Println("Error beginning transaction:", err)
		return
	}
	defer tx.Close(ctx)

	// Define your Cypher query
	query := `
		MATCH (r:Recipe)
		WHERE r.name = 'Peanut Sauce' OR r.name = 'Vietnamese Spring Rolls (Gỏi Cuốn)'
		WITH r
		MATCH (r)-[:CONTAINS]->(p:Product)
		OPTIONAL MATCH (p)-[:PURCHASE_AT]->(s:Store)
		WITH p, COLLECT(DISTINCT s) AS stores
		RETURN COLLECT(DISTINCT p.name) AS Ingredients,
			   [store IN stores | CASE WHEN store IS NOT NULL THEN store.name ELSE 'Unknown' END] AS Stores
		ORDER BY [store IN Stores | toLower(store)]
	`

	result, err := tx.Run(ctx, query, nil)
	if err != nil {
		fmt.Println("Error running Neo4j query:", err)
		return
	}

	var recipeIngredients []RecipeIngredient

	for result.Next(ctx) {
		record := result.Record()
		keys := record.Keys

		recipeIngredient := RecipeIngredient{}

		for _, key := range keys {
			value, _ := record.Get(key)

			// Check if the value is a list
			if list, ok := value.([]interface{}); ok {
				// If it's a list, loop over the elements of the list
				if key == "Ingredients" {
					recipeIngredient.Ingredients = make([]string, len(list))
					for i, item := range list {
						recipeIngredient.Ingredients[i] = item.(string)
					}
				} else if key == "Stores" {
					recipeIngredient.Stores = make([]string, len(list))
					for i, item := range list {
						recipeIngredient.Stores[i] = item.(string)
					}
				}
			}
		}

		recipeIngredients = append(recipeIngredients, recipeIngredient)
	}

	if err := result.Err(); err != nil {
		fmt.Println("Error during result iteration:", err)
		return
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		fmt.Println("Error committing transaction:", err)
		return
	}

	const outputTemplate = `
{{- range .Stores }}{{ . }}{{- end }}
{{- range .Ingredients }}
  - {{ . }}
{{- end }}
`

	tmpl := template.Must(template.New("output").Parse(outputTemplate))

	for i, recipe := range recipeIngredients {
		err := tmpl.Execute(os.Stdout, recipe)
		if err != nil {
			fmt.Println("Error executing template:", err)
			return
		}

		filename := fmt.Sprintf("recipe%d.txt", i+1)
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		err = tmpl.Execute(file, recipe)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
}
