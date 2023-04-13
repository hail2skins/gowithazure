// hotToCool.go is a simple program to gather all containers within a storage account and load them into a pager.
// The pager is then iterated over to gather all blobs within each container. If the blob is hot tier storage, it is
// changed to cool tier storage. This is a simple example of how to use the Azure SDK for Go to interact with Azure.
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
	config.ViperInit()
	auth.SetEnvCreds()
	url := viper.GetString("app.accounturl1")

	fmt.Printf("Azure Storage Account Container Count\n")
	fmt.Printf("Evaluating storage account %s\n", url)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)
	ctx := context.Background()
	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	pager := client.NewListContainersPager(&azblob.ListContainersOptions{})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, container := range resp.ContainerItems {
			containerClient := client.ServiceClient().NewContainerClient(*container.Name)
			blobPager := containerClient.NewListBlobsFlatPager(nil)

			for blobPager.More() {
				blobResp, err := blobPager.NextPage(ctx)
				if err != nil {
					break
				}
				for _, blob := range blobResp.Segment.BlobItems {
					if *blob.Properties.AccessTier == "Hot" {
						blockBlobClient := containerClient.NewBlockBlobClient(*blob.Name)
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
