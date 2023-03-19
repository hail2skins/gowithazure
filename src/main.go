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
	// Passing in viper setup config to get rolling from comfig\ViperInit file
	config.ViperInit()

	fmt.Printf("Azure Blob storage quick start sample\n")
	// see auth\azurelogin.go for function details
	auth.SetEnvCreds()

	// TODO: replace <storage-account-name> with your actual storage account name
	url := viper.GetString("app.accounturl")

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)
	ctx := context.Background()

	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	//Get a list of containers
	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: true},
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		utility.HandleError(err) // if err is not nil, break the loop.
		for _, container := range resp.ContainerItems {
			fmt.Printf("Container Name: %s\n", *container.Name)
		}
		total := len(resp.ContainerItems)

		fmt.Printf("There are %v containers in the storage account.", total)
	}

}
