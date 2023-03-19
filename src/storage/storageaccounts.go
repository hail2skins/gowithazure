package storage

import "github.com/spf13/viper"

// StorageAccounts is a slice of storage account URLs
var StorageAccounts = []string{
	viper.GetString("app.accounturl1"),
	viper.GetString("app.accounturl2"),
}

var TotalContainerCount int
var TotalBlobCount int
