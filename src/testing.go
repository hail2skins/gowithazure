package main

import (
	"context"
	"fmt"
	"gowithazure/src/auth"
	"gowithazure/src/config"

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

	// storageAccounts is a slice of strings containing the storage account URLs
	var storageAccounts = []string{
		viper.GetString("app.accounturldev"),
		//viper.GetString("app.accounturl2"),
	}

	var totalContainerCount int
	var totalBlobCount int

	for _, account := range storageAccounts {
		fmt.Printf("Storage account: %s\n", account)

		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			panic(err)
		}

		client, err := azblob.NewClient(account, cred, nil)
		if err != nil {
			panic(err)
		}

		containerPager := client.NewListContainersPager(nil)

		var containerCount int

		for containerPager.More() {
			page, err := containerPager.NextPage(context.Background())
			if err != nil {
				panic(err)
			}

			for _, container := range page.ContainerItems {
				containerCount++
				totalContainerCount++

				blobPager := client.NewListBlobsFlatPager(*container.Name, nil)

				var blobCount int
				for blobPager.More() {
					page, err := blobPager.NextPage(context.Background())
					if err != nil {
						panic(err)
					}

					blobCount += len(page.Segment.BlobItems)
					totalBlobCount += len(page.Segment.BlobItems)
				}

				//fmt.Printf("  Container: %s (%d blobs)\n", *container.Name, blobCount)
			}
		}
		fmt.Printf("  Container count: %d\n", containerCount)

		fmt.Println()
	}

	fmt.Printf("Total container count: %d\n", totalContainerCount)
	fmt.Printf("Total blob count: %d\n", totalBlobCount)
}
