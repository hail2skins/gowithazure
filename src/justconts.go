package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gowithazure/src/auth"
	"gowithazure/src/config"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	start := time.Now()

	// Initialize configuration and set environment variables for Azure authentication
	config.ViperInit()
	auth.SetEnvCreds()

	// Retrieve storage account URLs from configuration
	urls := []string{
		viper.GetString("app.accounturl2"),
		//viper.GetString("app.accounturl1"),
		// Add more URLs as needed
	}

	// Initialize a wait group to synchronize goroutines
	var wg sync.WaitGroup
	// Create a channel to communicate counts from goroutines
	countChannel := make(chan [2]int)

	for _, url := range urls {
		// Increment the wait group counter for each URL
		wg.Add(1)
		// Launch a goroutine for each URL
		go func(url string) {
			defer wg.Done() // Decrement the wait group counter when the goroutine completes
			total, empty := processURL(url)
			countChannel <- [2]int{total, empty} // Send the count to the channel
		}(url)
	}

	// Launch a goroutine to close the countChannel once all processing goroutines are done
	go func() {
		wg.Wait()           // Wait for all goroutines to finish
		close(countChannel) // Close the channel to signal completion
	}()

	// Aggregate counts from the channel
	totalContainers := 0
	totalEmptyContainers := 0

	for count := range countChannel {
		totalContainers += count[0]
		totalEmptyContainers += count[1]
	}

	// Output the total count and the time taken for processing
	fmt.Printf("Total containers across all accounts: %v\n", totalContainers)
	fmt.Printf("Total empty containers across all accounts: %v\n", totalEmptyContainers)
	fmt.Printf("Total time taken: %v\n", time.Since(start))
}

// processURL takes a storage account URL and returns the count of containers
func processURL(url string) (int, int) {
	// Create a default Azure credential object
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("Error creating credential: %v\n", err)
		return 0, 0 // Return 0s if there's an error creating the credential
	}

	// Create a context for the Azure SDK operations
	ctx := context.Background()

	// Create a client for the storage account
	client, err := azblob.NewClient(url, credential, nil)
	if err != nil {
		fmt.Printf("Error creating client for URL %s: %v\n", url, err)
		return 0, 0 // Return 0s if there's an error creating the client
	}

	// Initialize the pager for listing containers
	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: false},
	})

	// Count the containers and empty containers
	containerCount := 0
	emptyContainerCount := 0
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			fmt.Printf("Error getting next page for URL %s: %v\n", url, err)
			break // Exit the loop if there's an error getting the next page
		}
		containerCount += len(resp.ContainerItems)

		// Check each container for blobs
		for _, containerItem := range resp.ContainerItems {
			maxResults := int32(1) // Request only one blob to minimize data retrieval
			blobPager := client.NewListBlobsFlatPager(*containerItem.Name, &azblob.ListBlobsFlatOptions{
				Include:    azblob.ListBlobsInclude{Deleted: false},
				MaxResults: &maxResults,
			})

			// Fetch the first page of the blob listing
			blobResp, err := blobPager.NextPage(ctx)
			if err != nil {
				fmt.Printf("Error checking blobs in container %s: %v\n", *containerItem.Name, err)
				continue // Skip to next container on error
			}

			// Check if the page has any blobs
			if len(blobResp.Segment.BlobItems) == 0 {
				emptyContainerCount++
			}
		}
	}

	return containerCount, emptyContainerCount
}
