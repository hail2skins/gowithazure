// Description: Just list the containers in the storage account. This can engage a list of accounts
// It is important to know the way the SDK works it only returns 5000 items at a time. So if you have more than 5000 containers
// it takes a while.  The NewListContainersPager uses a marker interally. This is as fast as we get.
package main

import (
	"context"
	"fmt"
	"gowithazure/src/auth"
	"gowithazure/src/config"
	"gowithazure/src/utility"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	// Initialize the configuration file and get the viper object
	config.ViperInit()
	// see auth\azurelogin.go for function details. if using az login you can comment this out
	auth.SetEnvCreds()
	// config.yml has this and several other storage account urls to test with
	// Define a slice of strings containing the storage account urls to test with
	urls := []string{
		viper.GetString("app.accounturl1"),
		viper.GetString("app.accounturl2"),
		//viper.GetString("app.accounturl3"),
		// add more urls here as needed
	}

	// Iterate over the urls slice with a for loop
	for i, url := range urls {
		fmt.Printf("Azure Storage Account Container Count for %s\n", url)

		// Create a default credential object.  This will use the environment variables. Or variables from az login if you are using that.
		credential, err := azidentity.NewDefaultAzureCredential(nil)
		// Handle any errors that might have occurred
		utility.HandleError(err)
		// Create a context object
		ctx := context.Background()
		// Create a client object
		client, err := azblob.NewClient(url, credential, nil)
		utility.HandleError(err)
		// Get a list of containers
		pager := client.NewListContainersPager(&azblob.ListContainersOptions{
			Include: azblob.ListContainersInclude{Metadata: true, Deleted: false}, // Include metadata and exclude deleted containers
		})

		var totalContainers int
		// Loop through the pages of containers
		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				break
			}
			totalContainers += len(resp.ContainerItems)
		}

		fmt.Printf("There are %v containers in the storage account.\n", totalContainers)

		// Print a separator between each url's output
		if i < len(urls)-1 {
			fmt.Println("--------------------------------------------------")
		}
	}
}
