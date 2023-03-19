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
	// Passing in viper setup config to get rolling from config\ViperInit file
	config.ViperInit()

	// see auth\azurelogin.go for function details. Sets credentials.  If using az login, comment this out.
	auth.SetEnvCreds()

	// storageAccounts is a slice of strings containing the storage account URLs
	var storageAccounts = []string{
		viper.GetString("app.accounturl1"),
		viper.GetString("app.accounturl2"),
	}

	// totalContainerCount is a slice (array actually) of ints containing the total number of containers in each storage account
	var totalContainerCount [2]int
	// totalBlobCount is a slice (array actually) of ints containing the total number of blobs in each storage account
	var totalBlobCount [2]int

	// accountContainers is a slice of maps containing the names of the containers in each storage account
	accountContainers := make([]map[string]bool, 2)
	// accountBlobs is a slice of maps containing the names of the blobs in each storage account
	accountBlobs := make([]map[string]bool, 2)

	// Loop through the storage accounts
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

		// Get a list of containers
		containerPager := client.NewListContainersPager(nil)

		// Loop through the containers
		for containerPager.More() {
			page, err := containerPager.NextPage(context.Background())
			if err != nil {
				panic(err)
			}

			// Loop through the blobs in each container
			for _, container := range page.ContainerItems {
				totalContainerCount[i]++
				accountContainers[i][*container.Name] = true

				// Get a list of blobs in the container
				blobPager := client.NewListBlobsFlatPager(*container.Name, nil)

				// Loop through the blobs
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

	// Compare blobs between the two accounts
	for blobName := range accountBlobs[0] {
		if !accountBlobs[1][blobName] {
			fmt.Printf("Blob '%s' exists in primary storage account but has not been replicated\n", blobName)
		}
	}
}
