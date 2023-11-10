package cmd

import (
	"context"
	"fmt"
	"sort"

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

	result, err := neo4j.ExecuteQuery(ctx, driver, `
	MATCH (p:Product)
	WITH p.name AS productName, p.urls AS productUrls
	UNWIND productUrls AS url
	RETURN productName, url
	ORDER BY toLower(productName)
			`, map[string]any{}, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		panic(err)
	}

	products := make(map[string][]string)

	// Summary information
	fmt.Printf("\nThe query `%v` returned %v records in %+v.\n",
		result.Summary.Query().Text(), len(result.Records),
		result.Summary.ResultAvailableAfter())

	for _, record := range result.Records {
		fmt.Println(record.AsMap())
		productName, _ := record.Get("productName")
		url, _ := record.Get("url")

		products[productName.(string)] = append(products[productName.(string)], url.(string))
	}

	var productNames []string
	for key := range products {
		productNames = append(productNames, key)
	}
	sort.Strings(productNames)

	for _, name := range productNames {
		urls := products[name]
		sort.Strings(urls)
		fmt.Printf("%s:\n", name)
		for _, url := range urls {
			fmt.Printf("%s\n", url)
		}

		fmt.Println()
	}
	return nil
}
