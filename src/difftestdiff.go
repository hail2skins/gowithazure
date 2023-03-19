package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	// storageAccounts is a slice of strings containing the storage account URLs
	var storageAccounts = []string{
		viper.GetString("app.accounturl1"),
		viper.GetString("app.accounturl2"),
	}

	var totalContainerCount [2]int
	var totalBlobCount [2]int

	accountContainers := make([]map[string]bool, 2)
	accountBlobs := make([]map[string]bool, 2)

	for i, account := range storageAccounts {
		fmt.Printf("Storage account: %s\n", account)

		accountContainers[i] = make(map[string]bool)
		accountBlobs[i] = make(map[string]bool)

		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			panic(err)
		}

		client, err := azblob.NewClient(account, cred, nil)
		if err != nil {
			panic(err)
		}

		containerPager := client.NewListContainersPager(nil)

		for containerPager.More() {
			page, err := containerPager.NextPage(context.Background())
			if err != nil {
				panic(err)
			}

			for _, container := range page.ContainerItems {
				totalContainerCount[i]++
				accountContainers[i][*container.Name] = true

				blobPager := client.NewListBlobsFlatPager(*container.Name, nil)

				for blobPager.More() {
					page, err := blobPager.NextPage(context.Background())
					if err != nil {
						panic(err)
					}

					for _, blob := range page.Segment.BlobItems {
						totalBlobCount[i]++
						accountBlobs[i][*container.Name+"/"+*blob.Name] = true
					}
				}
			}
		}

		fmt.Printf("  Container count: %d\n", totalContainerCount[i])
		fmt.Printf("  Blob count: %d\n", totalBlobCount[i])
	}

	// Compare containers and blobs between the two accounts
	for containerName := range accountContainers[0] {
		if !accountContainers[1][containerName] {
			fmt.Printf("Container '%s' exists in first account but not in second\n", containerName)
		}
	}

	for blobName := range accountBlobs[0] {
		if !accountBlobs[1][blobName] {
			fmt.Printf("Blob '%s' exists in first account but not in second\n", blobName)
		}
	}
}
