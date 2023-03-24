package main

import (
	"context"
	"fmt"
	"gowithazure/src/auth"
	"gowithazure/src/config"
	"gowithazure/src/utility"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/spf13/viper"
)

func main() {
	config.ViperInit()
	auth.SetEnvCreds()
	url := viper.GetString("app.accounturldev")

	fmt.Printf("Azure Storage Account Container Count\n")
	fmt.Printf("Evaluating storage account %s\n", url)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	utility.HandleError(err)
	ctx := context.Background()
	client, err := azblob.NewClient(url, credential, nil)
	utility.HandleError(err)

	pager := client.NewListContainersPager(&azblob.ListContainersOptions{
		Include: azblob.ListContainersInclude{Metadata: true, Deleted: false},
	})

	var totalContainers int
	nameLengthCounts := make(map[int]int)
	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	thirtyDaysAgo := time.Now().AddDate(0, -1, 0)
	oldContainersCount := 0
	recentContainersCount := 0

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, container := range resp.ContainerItems {
			totalContainers++
			nameLength := len(*container.Name)
			nameLengthCounts[nameLength]++

			if container.Properties.LastModified.Before(twoYearsAgo) {
				oldContainersCount++
			}

			if container.Properties.LastModified.After(thirtyDaysAgo) {
				recentContainersCount++
			}
		}
	}

	fmt.Printf("There are %v containers in the storage account.\n", totalContainers)
	fmt.Println("Containers by name length:")
	for nameLen, count := range nameLengthCounts {
		fmt.Printf("  Containers with name length of %d characters: %d\n", nameLen, count)
	}
	fmt.Printf("Containers last modified more than two years ago: %d\n", oldContainersCount)
	fmt.Printf("Containers created within the last 30 days: %d\n", recentContainersCount)
}
