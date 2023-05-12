// Description: auProdVideoContainerEvaluation.go evaluates the 10 production storage for specific criteria.
// We are looking for files with the suffix -in or -out that have not been modified in the last 7 days.
// We list out the Totals for containers, -in suffix, -out suffix and both suffixes that have not been modified in the last 7 days.
// It is important to know the way the SDK works it only returns 5000 items at a time. So if you have more than 5000 containers
// it takes a while.  The NewListContainersPager uses a marker interally. This is as fast as we get.
package main

import (
	"context"
	"fmt"
	"gowithazure/src/auth"
	"gowithazure/src/config"
	"gowithazure/src/utility"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

// ContainerStats is a struct to hold statistics about a container.
type ContainerStats struct {
	Url                  string
	TotalContainers      int
	TotalInContainers    int
	TotalOutContainers   int
	TotalInOutContainers int
}

// processUrl is a goroutine for processing each URL.
// It updates the wait group and sends the results via the results channel.
func processUrl(url string, wg *sync.WaitGroup, results chan<- ContainerStats) {
	// Decrement the WaitGroup counter when the goroutine completes.
	defer wg.Done()

	// Create a default Azure credential.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)

	// Create a context for our operations.
	ctx := context.Background()

	// Create a new blob storage client.
	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	// Get a pager for listing the containers.
	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: false},
	})

	// Initialize the stats for this URL.
	var stats ContainerStats
	stats.Url = url

	// Loop over the pages of containers.
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			break
		}

		// Loop over the containers in the current page.
		for _, container := range resp.ContainerItems {
			// Only consider containers older than 7 days.
			if time.Since(*container.Properties.LastModified) > 7*24*time.Hour {
				name := container.Name
				stats.TotalContainers++
				if strings.HasSuffix(*name, "-in") {
					stats.TotalInContainers++
					stats.TotalInOutContainers++
				} else if strings.HasSuffix(*name, "-out") {
					stats.TotalOutContainers++
					stats.TotalInOutContainers++
				}
			}
		}
	}

	// Send the stats to the results channel.
	results <- stats
}

// main is the entry point of our script.
func main() {
	// Initialize the application configuration.
	config.AUProdViperInit()

	// Set up the Azure credentials.
	auth.SetEnvCreds()

	// List of URLs to process.
	urls := []string{
		viper.GetString("app.auprodaccounturl1"),
		viper.GetString("app.auprodaccounturl2"),
		viper.GetString("app.auprodaccounturl3"),
		viper.GetString("app.auprodaccounturl4"),
		viper.GetString("app.auprodaccounturl5"),
		viper.GetString("app.auprodaccounturl6"),
		viper.GetString("app.auprodaccounturl7"),
		viper.GetString("app.auprodaccounturl8"),
		viper.GetString("app.auprodaccounturl9"),
		viper.GetString("app.auprodaccounturl10"),
	}

	// Create a WaitGroup to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Create a channel to receive the results from the goroutines.
	results := make(chan ContainerStats, len(urls))

	// Launch a goroutine for each URL.
	for _, url := range urls {
		wg.Add(1)
		go processUrl(url, &wg, results)
	}

	// Launch a goroutine to close the results channel after all other goroutines finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Loop over the results channel, printing results as they arrive.
	for stats := range results {
		fmt.Printf("Azure Storage Account Container Count for containers not modified for 7 days %s\n", stats.Url)
		fmt.Printf("There are %v containers in the storage account.\n", stats.TotalContainers)
		fmt.Printf("There are %v containers with -in suffix in the storage account.\n", stats.TotalInContainers)
		fmt.Printf("There are %v containers with -out suffix in the storage account.\n", stats.TotalOutContainers)
		fmt.Printf("There are %v containers with either -in or -out suffix in the storage account.\n", stats.TotalInOutContainers)
		fmt.Println("--------------------------------------------------")
	}
}
