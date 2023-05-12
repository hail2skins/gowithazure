// hotToCool.go is a simple program that demonstrates how to use the Azure SDK for Go to interact with Azure.
// It gathers all containers within a storage account, iterates over them, and changes the access tier of hot blobs to cool.
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
	// Initialize configuration and set environment credentials.
	config.ViperInit()
	auth.SetEnvCreds()

	// Retrieve the storage account URL from the configuration.
	url := viper.GetString("app.accounturl1")

	fmt.Printf("Making it cool, from hot\n")
	fmt.Printf("Evaluating storage account %s\n", url)

	// Create a default Azure credential and a context.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)
	ctx := context.Background()

	// Create a new Azure Blob Storage client.
	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	// Create a pager to list all containers in the storage account.
	pager := client.NewListContainersPager(&azblob.ListContainersOptions{})

	// Iterate through each container in the pager.
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, container := range resp.ContainerItems {
			// Create a container client for the current container.
			containerClient := client.ServiceClient().NewContainerClient(*container.Name)

			// Create a pager to list all blobs in the current container.
			blobPager := containerClient.NewListBlobsFlatPager(nil)

			// Iterate through each blob in the pager.
			for blobPager.More() {
				blobResp, err := blobPager.NextPage(ctx)
				if err != nil {
					break
				}
				for _, blob := range blobResp.Segment.BlobItems {
					// Check if the blob's access tier is "Hot".
					if *blob.Properties.AccessTier == "Hot" {
						// Create a block blob client for the current blob.
						blockBlobClient := containerClient.NewBlockBlobClient(*blob.Name)

						// Change the blob's access tier from "Hot" to "Cool".
						_, err := blockBlobClient.SetTier(ctx, "Cool", nil)
						if err != nil {
							fmt.Println("Error setting blob tier:", err)
						} else {
							fmt.Printf("Successfully changed the access tier of '%s' to Cool\n", *blob.Name)
						}
					}
				}
			}
		}
	}
}
