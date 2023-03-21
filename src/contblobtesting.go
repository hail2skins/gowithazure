package main

import (
	"context"
	"fmt"
	"gowithazure/src/auth"
	"gowithazure/src/config"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

type AccountResult struct {
	accountURL     string
	containerCount int
	blobCount      int
}

type ContainerResult struct {
	containerName string
	blobCount     int
}

func processContainer(client *azblob.Client, containerName string, wg *sync.WaitGroup, results chan<- ContainerResult) {
	defer wg.Done()

	blobPager := client.NewListBlobsFlatPager(containerName, nil)

	var blobCount int
	for blobPager.More() {
		page, err := blobPager.NextPage(context.Background())
		if err != nil {
			panic(err)
		}

		blobCount += len(page.Segment.BlobItems)
	}

	results <- ContainerResult{containerName: containerName, blobCount: blobCount}
}

func processAccount(account string, wg *sync.WaitGroup, results chan<- AccountResult) {
	defer wg.Done()

	fmt.Printf("Processing storage account: %s\n", account)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}

	client, err := azblob.NewClient(account, credential, nil)
	if err != nil {
		panic(err)
	}

	containerPager := client.NewListContainersPager(nil)

	var containerCount int
	var containerResults = make(chan ContainerResult)

	var containerWg sync.WaitGroup

	for containerPager.More() {
		page, err := containerPager.NextPage(context.Background())
		if err != nil {
			panic(err)
		}

		containerCount += len(page.ContainerItems)

		for _, container := range page.ContainerItems {
			containerWg.Add(1)
			go processContainer(client, *container.Name, &containerWg, containerResults)
		}
	}

	go func() {
		containerWg.Wait()
		close(containerResults)
	}()

	var blobCount int

	for result := range containerResults {
		fmt.Printf("  Container: %s\n", result.containerName)
		fmt.Printf("    Blob count: %d\n", result.blobCount)

		blobCount += result.blobCount
	}

	results <- AccountResult{accountURL: account, containerCount: containerCount, blobCount: blobCount}
}

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

	results := make(chan AccountResult)
	var wg sync.WaitGroup

	for _, account := range storageAccounts {
		wg.Add(1)
		go processAccount(account, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		//fmt.Printf("Storage account: %s\n", result.accountURL)
		//fmt.Printf("  Container count: %d\n", result.containerCount)
		//fmt.Printf("  Blob count: %d\n", result.blobCount)
		//fmt.Println()

		totalContainerCount += result.containerCount
		totalBlobCount += result.blobCount
	}

	fmt.Printf("Total container count: %d\n", totalContainerCount)
	fmt.Printf("Total blob count: %d\n", totalBlobCount)
}
