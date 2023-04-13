package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func main() {
	srcAccountURL := "https://goteststorageaccount.blob.core.windows.net/"
	dstAccountURL := "https://goteststoragesecondary.blob.core.windows.net/"
	srcContainerName := "nutty"
	dstContainerName := "nutty"
	blobName := "Proof_of_Insurance.pdf"

	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}

	srcClient, err := azblob.NewClient(srcAccountURL, credential, nil)
	if err != nil {
		log.Fatal(err)
	}

	dstClient, err := azblob.NewClient(dstAccountURL, credential, nil)
	if err != nil {
		log.Fatal(err)
	}

	srcContainerClient := srcClient.ServiceClient().NewContainerClient(srcContainerName)
	dstContainerClient := dstClient.ServiceClient().NewContainerClient(dstContainerName)

	createResp, err := dstContainerClient.Create(ctx, nil)
	if err != nil {
		log.Printf("Failed to create container, it might already exist: %v", err)
	} else {
		log.Printf("Container created successfully, ETag: %s", *createResp.ETag)
	}

	srcBlobClient := srcContainerClient.NewBlobClient(blobName)
	dstBlobClient := dstContainerClient.NewBlobClient(blobName)

	copySource := srcBlobClient.URL()
	resp, err := dstBlobClient.StartCopyFromURL(ctx, copySource, nil)
	if err != nil {
		log.Fatalf("Failed to copy blob: %v", err)
	}

	fmt.Printf("Blob copy started, copy ID: %s\n", *resp.CopyID)
}
