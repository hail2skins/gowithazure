package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func main() {
	fmt.Printf("Azure Blob storage quick start sample\n")

	// TODO: replace <storage-account-name> with your actual storage account name
	url := "https://goteststorageaccount.blob.core.windows.net/"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	HandleError(err)
	ctx := context.Background()

	client, err := azblob.NewClient(url, credential, nil)
	HandleError(err)

	//Get a list of containers
	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: true},
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		HandleError(err) // if err is not nil, break the loop.
		for _, container := range resp.ContainerItems {
			fmt.Printf("Container Name: %s\n", *container.Name)
		}
		fmt.Println(len(resp.ContainerItems))
	}

}
