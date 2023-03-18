package main

import (
	"context"
	"fmt"
	"gowithazure/src/utility"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("../")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	fmt.Printf("Azure Blob storage quick start sample\n")
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
		fmt.Println(len(resp.ContainerItems))
	}

}
