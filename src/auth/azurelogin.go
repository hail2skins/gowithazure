package auth

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// SetCreds presumes you have an Azure Service Principal account setup
// and you've been provided the credentials with the appropriate permissions
// set within Azure AD. You can comment out the call to auth.SetEnvCreds in
// main.go if you simply wish to use the az login cli default creds to test.
func SetEnvCreds() {
	os.Setenv("AZURE_TENANT_ID", viper.GetString("app.AZURE_TENANT_ID"))
	os.Setenv("AZURE_CLIENT_ID", viper.GetString("app.AZURE_CLIENT_ID"))
	os.Setenv("AZURE_CLIENT_SECRET", viper.GetString("app.AZURE_CLIENT_SECRET"))
	fmt.Println("Setting environment variables")

	// // Additional print statements to verify the environment variables
	// fmt.Println("AZURE_TENANT_ID:", os.Getenv("AZURE_TENANT_ID"))
	// fmt.Println("AZURE_CLIENT_ID:", os.Getenv("AZURE_CLIENT_ID"))
	// fmt.Println("AZURE_CLIENT_SECRET:", os.Getenv("AZURE_CLIENT_SECRET"))
}
