package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/olekukonko/tablewriter"
)

// VM represents a virtual machine with its subscription context
type VM struct {
	Name              string
	ResourceGroup     string
	Location          string
	SubscriptionID    string
	SubscriptionName  string
	VMSize            string
	OSType            string
	PrivateIPs        []string
	PublicIPs         []string
	OSName            string
	OSVersion         string
	Tags              map[string]string
	AdminUsername     string
	NetworkInterfaces []string
	AvailabilitySet   string
	DataDisks         []string
	BootDiagnostics   string
}

// Subscription represents an Azure subscription
type Subscription struct {
	ID   string
	Name string
}

func main() {
	// Create a context for the API calls with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Authenticate to Azure
	fmt.Println("Authenticating to Azure...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Failed to obtain Azure credential: %v", err)
	}

	// Create a client for subscription operations
	subClient, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		log.Fatalf("Failed to create subscription client: %v", err)
	}

	// Get all subscriptions
	fmt.Println("Fetching all subscriptions...")
	pager := subClient.NewListPager(nil)

	// Store all subscriptions
	var allSubscriptions []Subscription

	// Iterate through all subscriptions
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("Failed to get subscriptions: %v", err)
		}

		for _, sub := range page.Value {
			allSubscriptions = append(allSubscriptions, Subscription{
				ID:   *sub.SubscriptionID,
				Name: *sub.DisplayName,
			})
		}
	}

	// Display subscriptions and let user select
	selectedSubscriptions := selectSubscriptions(allSubscriptions)

	if len(selectedSubscriptions) == 0 {
		fmt.Println("No subscriptions selected. Exiting.")
		return
	}

	var allVMs []VM

	// Ask if user wants to collect detailed network information
	fmt.Println("\nCollect detailed network information? This may take longer. (y/n):")
	reader := bufio.NewReader(os.Stdin)
	collectNetworkInfo, _ := reader.ReadString('\n')
	collectNetworkInfo = strings.TrimSpace(collectNetworkInfo)
	collectNetworkInfo = strings.ToLower(collectNetworkInfo)
	getDetailedNetworkInfo := collectNetworkInfo == "y" || collectNetworkInfo == "yes"

	// Process selected subscriptions
	for _, sub := range selectedSubscriptions {
		fmt.Printf("Processing subscription: %s (%s)\n", sub.Name, sub.ID)

		// Create a client for VM operations in this subscription
		vmClient, err := armcompute.NewVirtualMachinesClient(sub.ID, cred, nil)
		if err != nil {
			log.Printf("Failed to create VM client for subscription %s: %v", sub.ID, err)
			continue
		}

		// List all VMs in the subscription
		vmPager := vmClient.NewListAllPager(nil)

		for vmPager.More() {
			fmt.Println("Fetching next page of VMs...")
			vmPage, err := vmPager.NextPage(ctx)
			if err != nil {
				log.Printf("Failed to get VMs for subscription %s: %v", sub.ID, err)
				break
			}

			fmt.Printf("Found %d VMs in this page\n", len(vmPage.Value))
			for i, virtualMachine := range vmPage.Value {
				fmt.Printf("Processing VM %d/%d: %s\n", i+1, len(vmPage.Value), *virtualMachine.Name)

				vm := VM{
					Name:             *virtualMachine.Name,
					ResourceGroup:    extractResourceGroup(*virtualMachine.ID),
					Location:         *virtualMachine.Location,
					SubscriptionID:   sub.ID,
					SubscriptionName: sub.Name,
					Tags:             make(map[string]string),
				}

				// Get VM size
				if virtualMachine.Properties != nil && virtualMachine.Properties.HardwareProfile != nil && virtualMachine.Properties.HardwareProfile.VMSize != nil {
					vm.VMSize = string(*virtualMachine.Properties.HardwareProfile.VMSize)
				}

				// Get OS type
				if virtualMachine.Properties != nil && virtualMachine.Properties.StorageProfile != nil && virtualMachine.Properties.StorageProfile.OSDisk != nil && virtualMachine.Properties.StorageProfile.OSDisk.OSType != nil {
					vm.OSType = string(*virtualMachine.Properties.StorageProfile.OSDisk.OSType)
				}

				// Get OS details
				if virtualMachine.Properties != nil && virtualMachine.Properties.StorageProfile != nil &&
					virtualMachine.Properties.StorageProfile.ImageReference != nil {
					imgRef := virtualMachine.Properties.StorageProfile.ImageReference

					if imgRef.Offer != nil {
						vm.OSName = *imgRef.Offer
					}

					if imgRef.SKU != nil {
						vm.OSVersion = *imgRef.SKU
					}

					// Combine publisher and offer for a more complete OS name
					if imgRef.Publisher != nil {
						vm.OSName = *imgRef.Publisher + ":" + vm.OSName
					}
				}

				// Get availability set if available
				if virtualMachine.Properties != nil && virtualMachine.Properties.AvailabilitySet != nil &&
					virtualMachine.Properties.AvailabilitySet.ID != nil {
					vm.AvailabilitySet = *virtualMachine.Properties.AvailabilitySet.ID
				}

				// Get data disks
				if virtualMachine.Properties != nil && virtualMachine.Properties.StorageProfile != nil &&
					virtualMachine.Properties.StorageProfile.DataDisks != nil {
					for _, disk := range virtualMachine.Properties.StorageProfile.DataDisks {
						if disk.Name != nil {
							vm.DataDisks = append(vm.DataDisks, *disk.Name)
						}
					}
				}

				// Get boot diagnostics status
				if virtualMachine.Properties != nil && virtualMachine.Properties.DiagnosticsProfile != nil &&
					virtualMachine.Properties.DiagnosticsProfile.BootDiagnostics != nil {
					if virtualMachine.Properties.DiagnosticsProfile.BootDiagnostics.Enabled != nil {
						if *virtualMachine.Properties.DiagnosticsProfile.BootDiagnostics.Enabled {
							vm.BootDiagnostics = "Enabled"
						} else {
							vm.BootDiagnostics = "Disabled"
						}
					}
				}

				// Get tags
				if virtualMachine.Tags != nil {
					for k, v := range virtualMachine.Tags {
						if v != nil {
							vm.Tags[k] = *v
						} else {
							vm.Tags[k] = ""
						}
					}
				}

				// Get admin username
				if virtualMachine.Properties != nil && virtualMachine.Properties.OSProfile != nil && virtualMachine.Properties.OSProfile.AdminUsername != nil {
					vm.AdminUsername = *virtualMachine.Properties.OSProfile.AdminUsername
				}

				// Get VM ID
				if virtualMachine.ID != nil {
					// We don't store VMId anymore
				}

				// Get power state
				vmInstanceViewCtx, vmInstanceViewCancel := context.WithTimeout(ctx, 10*time.Second)
				defer vmInstanceViewCancel()

				vmInstanceView, err := vmClient.InstanceView(vmInstanceViewCtx, vm.ResourceGroup, vm.Name, nil)
				if err != nil {
					fmt.Printf("Warning: Failed to get instance view for VM %s: %v\n", vm.Name, err)
				} else if vmInstanceView.Statuses != nil {
					for _, status := range vmInstanceView.Statuses {
						if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState") {
							// We don't store PowerState anymore
							break
						}
					}
				}

				// Only get network info if user opted for it
				if getDetailedNetworkInfo {
					// Get network interfaces and IP addresses with a timeout
					fmt.Printf("Getting network info for VM: %s\n", vm.Name)
					err = getVMNetworkInfo(ctx, cred, &vm, virtualMachine)
					if err != nil {
						log.Printf("Warning: Failed to get network info for VM %s: %v", vm.Name, err)
					}
				} else {
					// Just collect the network interface IDs without detailed info
					if virtualMachine.Properties != nil && virtualMachine.Properties.NetworkProfile != nil &&
						virtualMachine.Properties.NetworkProfile.NetworkInterfaces != nil {
						for _, nicRef := range virtualMachine.Properties.NetworkProfile.NetworkInterfaces {
							if nicRef.ID != nil {
								vm.NetworkInterfaces = append(vm.NetworkInterfaces, *nicRef.ID)
							}
						}
					}
				}

				// Extract just the name from network interfaces
				for i, nic := range vm.NetworkInterfaces {
					parts := strings.Split(nic, "/")
					if len(parts) > 0 {
						vm.NetworkInterfaces[i] = parts[len(parts)-1]
					}
				}

				// Extract just the name from availability set
				if vm.AvailabilitySet != "" {
					parts := strings.Split(vm.AvailabilitySet, "/")
					if len(parts) > 0 {
						vm.AvailabilitySet = parts[len(parts)-1]
					}
				}

				allVMs = append(allVMs, vm)
				fmt.Printf("Completed processing VM: %s\n", vm.Name)
			}
		}
	}

	fmt.Printf("Processing complete. Found %d VMs total.\n", len(allVMs))

	// Sort VMs by name in descending order
	sort.Slice(allVMs, func(i, j int) bool {
		return allVMs[i].Name > allVMs[j].Name
	})

	// Print the results with improved formatting
	printVMTable(allVMs)

	// Export to CSV
	exportToCSV(allVMs)
}

// printVMTable prints the VM list in a clean, formatted table
func printVMTable(vms []VM) {
	// Sort VMs by name in descending order
	sort.Slice(vms, func(i, j int) bool {
		return vms[i].Name > vms[j].Name
	})

	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Name", "Resource Group", "Location", "Subscription", "VM Size", "OS Type",
		"OS Name", "OS Version", "Admin Username", "Network Interfaces", "Availability Set",
	})

	// Add VM data to the table
	for _, vm := range vms {
		// Join network interfaces and IPs with commas
		networkInterfaces := strings.Join(vm.NetworkInterfaces, ", ")

		table.Append([]string{
			vm.Name,
			vm.ResourceGroup,
			vm.Location,
			vm.SubscriptionName,
			vm.VMSize,
			vm.OSType,
			vm.OSName,
			vm.OSVersion,
			vm.AdminUsername,
			networkInterfaces,
			vm.AvailabilitySet,
		})
	}

	// Print the table
	fmt.Println("\nSummary of all virtual machines (sorted by name in descending order):")
	table.Render()
}

// selectSubscriptions displays all subscriptions and lets the user select which ones to process
func selectSubscriptions(subscriptions []Subscription) []Subscription {
	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Subscription Name", "Subscription ID"})

	// Add subscription data to the table
	for i, sub := range subscriptions {
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			sub.Name,
			sub.ID,
		})
	}

	// Print the table
	fmt.Println("\nAvailable Subscriptions:")
	table.Render()

	fmt.Println("\nSelect subscriptions to process:")
	fmt.Println("  - Enter comma-separated numbers (e.g., \"1,3,5\") for specific subscriptions")
	fmt.Println("  - Enter \"a\" for all subscriptions")
	fmt.Println("  - Enter \"q\" to quit")
	fmt.Print("\nYour selection: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Check for quit
	if input == "q" || input == "Q" {
		return nil
	}

	// Check for all
	if input == "a" || input == "A" {
		return subscriptions
	}

	// Process individual selections
	selected := []Subscription{}
	selections := strings.Split(input, ",")

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		idx, err := strconv.Atoi(sel)
		if err != nil {
			fmt.Printf("Invalid selection: %s (skipping)\n", sel)
			continue
		}

		// Adjust for 1-based indexing
		idx--

		if idx >= 0 && idx < len(subscriptions) {
			selected = append(selected, subscriptions[idx])
		} else {
			fmt.Printf("Selection out of range: %s (skipping)\n", sel)
		}
	}

	return selected
}

// extractResourceGroup extracts the resource group name from the VM ID
func extractResourceGroup(vmID string) string {
	// VM ID format: /subscriptions/{subID}/resourceGroups/{resourceGroup}/providers/...
	parts := strings.Split(vmID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

// getVMNetworkInfo retrieves network interfaces and IP addresses for a VM
func getVMNetworkInfo(ctx context.Context, cred *azidentity.DefaultAzureCredential, vm *VM, vmResource *armcompute.VirtualMachine) error {
	// Skip if no network interfaces
	if vmResource.Properties == nil || vmResource.Properties.NetworkProfile == nil || vmResource.Properties.NetworkProfile.NetworkInterfaces == nil {
		return nil
	}

	// Create a client for network interfaces
	nicClient, err := armnetwork.NewInterfacesClient(vm.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create network interface client: %w", err)
	}

	// Process each network interface
	for _, nicRef := range vmResource.Properties.NetworkProfile.NetworkInterfaces {
		if nicRef.ID == nil {
			continue
		}

		// Extract resource group and NIC name from the ID
		nicID := *nicRef.ID
		vm.NetworkInterfaces = append(vm.NetworkInterfaces, nicID)

		// Extract parts from the NIC ID
		parts := strings.Split(nicID, "/")
		if len(parts) < 9 {
			continue
		}

		resourceGroup := parts[4]
		nicName := parts[8]

		// Get the network interface details
		nic, err := nicClient.Get(ctx, resourceGroup, nicName, nil)
		if err != nil {
			return fmt.Errorf("failed to get network interface %s: %w", nicName, err)
		}

		// Process IP configurations
		if nic.Properties == nil || nic.Properties.IPConfigurations == nil {
			continue
		}

		for _, ipConfig := range nic.Properties.IPConfigurations {
			if ipConfig.Properties == nil {
				continue
			}

			// Get private IP address
			if ipConfig.Properties.PrivateIPAddress != nil {
				vm.PrivateIPs = append(vm.PrivateIPs, *ipConfig.Properties.PrivateIPAddress)
			}

			// Get public IP address if available
			if ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != nil {
				publicIPID := *ipConfig.Properties.PublicIPAddress.ID
				parts := strings.Split(publicIPID, "/")
				if len(parts) < 9 {
					continue
				}

				resourceGroup := parts[4]
				publicIPName := parts[8]

				// Create a client for public IPs
				publicIPClient, err := armnetwork.NewPublicIPAddressesClient(vm.SubscriptionID, cred, nil)
				if err != nil {
					return fmt.Errorf("failed to create public IP client: %w", err)
				}

				// Get the public IP details
				publicIP, err := publicIPClient.Get(ctx, resourceGroup, publicIPName, nil)
				if err != nil {
					return fmt.Errorf("failed to get public IP %s: %w", publicIPName, err)
				}

				if publicIP.Properties != nil && publicIP.Properties.IPAddress != nil {
					vm.PublicIPs = append(vm.PublicIPs, *publicIP.Properties.IPAddress)
				}
			}
		}
	}

	return nil
}

// extractResourceName extracts a resource name from its Azure resource ID
func extractResourceName(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	return parts[len(parts)-1]
}

// exportToCSV exports VM data to a CSV file
func exportToCSV(vms []VM) {
	fmt.Println("\nExporting VM data to CSV...")

	if len(vms) == 0 {
		fmt.Println("No VMs to export.")
		return
	}

	// Create a timestamp for the filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("azure_vms_%s.csv", timestamp)

	// Create the CSV file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	header := []string{
		"Name", "Resource Group", "Location", "Subscription ID", "Subscription Name",
		"VM Size", "OS Type", "OS Name", "OS Version", "Admin Username",
		"Private IPs", "Public IPs", "Network Interfaces", "Availability Set",
		"Data Disks", "Boot Diagnostics", "Tags",
	}

	if err := writer.Write(header); err != nil {
		fmt.Printf("Error writing CSV header: %v\n", err)
		return
	}

	// Track successful writes
	successCount := 0

	// Write VM data
	for _, vm := range vms {
		fmt.Printf("Writing VM to CSV: %s\n", vm.Name)

		// Convert slices to comma-separated strings
		privateIPs := strings.Join(vm.PrivateIPs, ", ")
		publicIPs := strings.Join(vm.PublicIPs, ", ")
		networkInterfaces := strings.Join(vm.NetworkInterfaces, ", ")
		dataDisks := strings.Join(vm.DataDisks, ", ")

		// Convert tags map to string
		var tagsStr string
		for k, v := range vm.Tags {
			tagsStr += fmt.Sprintf("%s:%s; ", k, v)
		}
		tagsStr = strings.TrimSuffix(tagsStr, "; ")

		// Create the record
		record := []string{
			vm.Name,
			vm.ResourceGroup,
			vm.Location,
			vm.SubscriptionID,
			vm.SubscriptionName,
			vm.VMSize,
			vm.OSType,
			vm.OSName,
			vm.OSVersion,
			vm.AdminUsername,
			privateIPs,
			publicIPs,
			networkInterfaces,
			vm.AvailabilitySet,
			dataDisks,
			vm.BootDiagnostics,
			tagsStr,
		}

		// Write the record
		if err := writer.Write(record); err != nil {
			fmt.Printf("Error writing VM %s to CSV: %v\n", vm.Name, err)
			continue
		}

		successCount++
	}

	// Flush the writer to ensure all data is written
	writer.Flush()
	if err := writer.Error(); err != nil {
		fmt.Printf("Error flushing CSV writer: %v\n", err)
	}

	// Get absolute path for better user feedback
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename // Fallback to relative path
	}

	fmt.Printf("\nSuccessfully exported %d/%d VMs to CSV file: %s\n", successCount, len(vms), absPath)
}

// printSubscriptionTable prints the subscription list in a clean, formatted table
func printSubscriptionTable(subs []Subscription) {
	if len(subs) == 0 {
		fmt.Println("\nNo subscriptions found.")
		return
	}

	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Name", "ID"})

	// Add subscription data to the table
	for i, sub := range subs {
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			sub.Name,
			sub.ID,
		})
	}

	// Print the table
	fmt.Println("\nAvailable subscriptions:")
	table.Render()
}

// printSubscriptions prints the list of subscriptions
func printSubscriptions(subscriptions []Subscription) {
	if len(subscriptions) == 0 {
		fmt.Println("\nNo subscriptions found.")
		return
	}

	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Subscription Name", "Subscription ID"})

	// Add subscription data to the table
	for i, sub := range subscriptions {
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			sub.Name,
			sub.ID,
		})
	}

	// Print the table
	fmt.Println("\nAvailable subscriptions:")
	table.Render()
}
