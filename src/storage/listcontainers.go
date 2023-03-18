package storage

import (
	"context"
	"fmt"
	"gowithazure/utility"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func main() {
	fmt.Printf("Azure Storage Account list and count containers\n")

	// TODO: replace <storage-account-name> with your actual storage account name
	url := "https://goteststorageaccount.blob.core.windows.net/"

	credential, err := azidentity.NewDefaultAzureCredential(nil) //use AZ Login default credential provided by azidentity package
	utility.HandleError(err)                                     //call to utility package for error function
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