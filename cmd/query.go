package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

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

type Data struct {
	Result struct {
		Ingredients []struct {
		} `json:"Ingredients"`
	} `json:"result"`
}

type RecipeIngredient struct {
	Ingredients any
}

func test() error {
	ctx := context.Background()
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
		return err
	}
	defer tx.Close(ctx)

	recipeNames := []string{"Peanut Sauce", "Vietnamese Spring Rolls (Gỏi Cuốn)"}
	recipeNames = []string{"Vietnamese Spring Rolls (Gỏi Cuốn)"}

	queryTemplate := `
	MATCH (r:Recipe)
	WHERE r.name IN [{{ range $i, $name := . }}{{ if $i }}, {{ end }}'{{ $name }}'{{ end }}]
	WITH r
	MATCH (r)-[:CONTAINS]->(p:Product)
	OPTIONAL MATCH (p)-[:PURCHASE_AT]->(s:Store)
	WITH p, COLLECT(DISTINCT s) AS stores
	WITH p, stores, COLLECT(DISTINCT {name: p.name, urls: p.urls}) AS Ingredients,
		 [store IN stores | CASE WHEN store IS NOT NULL THEN store.name ELSE 'Unknown' END] AS Stores
	RETURN apoc.convert.toJson({
		Ingredients: Ingredients,
		Stores: Stores
	}) AS result
	ORDER BY [store IN Stores | toLower(store)]
	`

	tmpl := template.Must(template.New("query").Parse(queryTemplate))

	var query strings.Builder
	err = tmpl.Execute(&query, recipeNames)
	if err != nil {
		fmt.Println("Error constructing query:", err)
		return err
	}

	fmt.Println(query.String())

	result, _ := neo4j.ExecuteQuery(ctx, driver, query.String(),
		map[string]any{}, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	for _, record := range result.Records {
		fmt.Println(record.AsMap())
	}

	for _, record := range result.Records {
		// Convert the map to a JSON byte array
		_, err := json.Marshal(record.AsMap())
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		value, found := record.Get("result")
		if !found {
			continue
		}

		var recipeIngredient RecipeIngredient
		if err := json.Unmarshal([]byte(value.(string)), &recipeIngredient); err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return err
		}

		stuff, _ := json.MarshalIndent(recipeIngredient, "", " ")
		fmt.Println(string(stuff))

	}

	return nil
}
