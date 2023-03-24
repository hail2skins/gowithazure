package main

import (
	"context"
	"fmt"
	"gowithazure/src/config"
	"gowithazure/src/utility"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	config.ViperInit()
	//auth.SetEnvCreds()
	url := viper.GetString("app.accounturlusprod")

	fmt.Printf("Azure Storage Account Container Count\n")
	fmt.Printf("Evaluating storage account %s\n", url)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)
	ctx := context.Background()
	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: true},
	})

	containerNames := make(map[string]bool)
	containerChan := make(chan []string)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pager.More() {
				resp, err := pager.NextPage(ctx)
				if err != nil {
					break
				}
				var containerItems []string
				for _, containerItem := range resp.ContainerItems {
					containerItems = append(containerItems, *containerItem.Name)
				}
				containerChan <- containerItems
			}
		}()
	}

	go func() {
		wg.Wait()
		close(containerChan)
	}()

	for containerItems := range containerChan {
		for _, name := range containerItems {
			if !containerNames[name] {
				containerNames[name] = true
			}
		}
	}

	fmt.Printf("There are %v containers in the storage account.\n", len(containerNames))
}
