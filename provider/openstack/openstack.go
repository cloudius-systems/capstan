package openstack

import (
	"fmt"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	glanceImages "github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
	"math"
	"os"
)

// GetOrPickFlavor returns flavors.Flavor struct that best matches arguments.
// First it decides whether user wanted to get flavor by name or by criteria.
// In case of flavor name, it retrieves data about that flavor and returns it.
// In case of criteria (diskMB, memoryMB) it returns smallest flavor that meets all criteria.
func GetOrPickFlavor(clientNova *gophercloud.ServiceClient, flavorName string, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if flavorName != "" {
		return GetFlavor(clientNova, flavorName, verbose)
	}
	return PickFlavor(clientNova, diskMB, memoryMB, verbose)
}

// PickFlavor picks flavor that best matches criteria (i.e. HDD size and RAM size).
// While diskMB is required, memoryMB is optional (set to -1 to ignore).
func PickFlavor(clientNova *gophercloud.ServiceClient, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if diskMB <= 0 {
		return nil, fmt.Errorf("Please specify disk size.")
	}

	var flavs []flavors.Flavor = listFlavors(clientNova, int(math.Ceil(float64(diskMB)/1024)), int(memoryMB))

	// Find smallest flavor for given conditions.
	if verbose {
		fmt.Printf("Find smallest flavor for conditions: diskMB >= %d AND memoryMB >= %d\n", diskMB, memoryMB)
	}
	var bestFlavor flavors.Flavor
	var minDiffDisk int64 = -1
	var minDiffMem int64 = -1
	for _, f := range flavs {
		diffDisk := int64(f.Disk)*1024 - diskMB
		var diffMem int64 = 0 // 0 is best value
		if memoryMB > 0 {
			diffMem = int64(f.RAM) - memoryMB
		}

		if diffDisk >= 0 && // disk is big enough
			(minDiffDisk == -1 || minDiffDisk > diffDisk) && // disk is smaller than current best, but still big enough
			diffMem >= 0 && // memory is big enough
			(minDiffMem == -1 || minDiffMem >= diffMem) { // memory is smaller than current best, but still big enough
			bestFlavor, minDiffDisk, minDiffMem = f, diffDisk, diffMem
		}
	}
	if minDiffDisk == -1 {
		return nil, fmt.Errorf("No flavor fits required conditions: diskMB >= %d AND memoryMB >= %d\n", diskMB, memoryMB)
	}
	return &bestFlavor, nil
}

// GetFlavor returns flavors.Flavor struct for given flavorName.
func GetFlavor(clientNova *gophercloud.ServiceClient, flavorName string, verbose bool) (*flavors.Flavor, error) {
	flavorId, err := flavors.IDFromName(clientNova, flavorName)
	if err != nil {
		return nil, err
	}

	flavor, err := flavors.Get(clientNova, flavorId).Extract()
	if err != nil {
		return nil, err
	}

	return flavor, nil
}

// PushImage first creates meta for image at OpenStack, then it sends binary data for it, the qcow2 image.
func PushImage(clientGlance *gophercloud.ServiceClient, imageName string, imageFilepath string, flavor *flavors.Flavor, verbose bool) {
	// Create metadata (on OpenStack).
	imgId, _ := createImage(clientGlance, imageName, flavor, verbose)
	// Send the image binary data to OpenStack
	uploadImage(clientGlance, imgId, imageFilepath, verbose)
}

// LaunchInstances launches <count> instances. Return first error that occurs or nil on success.
func LaunchInstances(clientNova *gophercloud.ServiceClient, name string, imageName string, flavorName string, count int, verbose bool) error {
	var err error
	if count <= 1 {
		// Take name as it is.
		err = launchServer(clientNova, name, flavorName, imageName, verbose)
	} else {
		// Append index after the name of each instance.
		for i := 0; i < count; i++ {
			currErr := launchServer(clientNova, fmt.Sprintf("%s-%d", name, (i+1)), flavorName, imageName, verbose)
			if err == nil {
				err = currErr
			}
		}
	}

	return err
}

// GetImageMeta returns images.Image struct for given imageName.
func GetImageMeta(clientNova *gophercloud.ServiceClient, imageName string, verbose bool) (*images.Image, error) {
	imageId, err := images.IDFromName(clientNova, imageName)
	if err != nil {
		return nil, err
	}

	image, err := images.Get(clientNova, imageId).Extract()
	if err != nil {
		return nil, err
	}

	return image, nil
}

/*
* Nova
 */

// listFlavors returns list of all flavors.
func listFlavors(clientNova *gophercloud.ServiceClient, minDiskGB int, minMemoryMB int) []flavors.Flavor {
	var flavs []flavors.Flavor = make([]flavors.Flavor, 0)

	pagerFlavors := flavors.ListDetail(clientNova, flavors.ListOpts{
		MinDisk: minDiskGB,
		MinRAM:  minMemoryMB,
	})
	pagerFlavors.EachPage(func(page pagination.Page) (bool, error) {
		flavorList, _ := flavors.ExtractFlavors(page)

		for _, f := range flavorList {
			flavs = append(flavs, f)
		}

		return true, nil
	})
	return flavs
}

// launchServer launches single server of given image.
func launchServer(clientNova *gophercloud.ServiceClient, name string, flavorName string, imageName string, verbose bool) error {
	resp := servers.Create(clientNova, servers.CreateOpts{
		Name:       name,
		FlavorName: flavorName,
		ImageName:  imageName,
	})
	if verbose {
		instance, err := resp.Extract()
		if err != nil {
			fmt.Println("Unable to create instance: %s", err)
		} else {
			fmt.Printf("Instance ID: %s\n", instance.ID)
		}
	}

	return resp.Err
}

/*
* Glance
 */

// createImage creates image metadata on OpenStack.
func createImage(clientGlance *gophercloud.ServiceClient, name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	createdImage, err := glanceImages.Create(clientGlance, glanceImages.CreateOpts{
		Name:             name,
		Tags:             []string{"tagOSv", "tagCapstan"},
		DiskFormat:       "qcow2",
		ContainerFormat:  "bare",
		MinDiskGigabytes: flavor.Disk,
		//MinRAMMegabytes: flavor.RAM  // TODO: Does it make sense to lock RAM during push??
	}).Extract()

	if err == nil && verbose {
		fmt.Printf("Created image [name: %s, ID: %s]\n", createdImage.Name, createdImage.ID)
	}
	return createdImage.ID, err
}

// uploadImage uploads image binary data to existing OpenStack image metadata.
func uploadImage(clientGlance *gophercloud.ServiceClient, imageId string, filepath string, verbose bool) error {
	if verbose {
		fmt.Printf("Uploading composed image '%s' to OpenStack\n", filepath)
	}

	f, err := os.Open(filepath)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	res := glanceImages.Upload(clientGlance, imageId, f)
	return res.Err
}

// deleteImage deletes image from OpenStack.
func deleteImage(clientGlance *gophercloud.ServiceClient, imageId string) {
	glanceImages.Delete(clientGlance, imageId)
}
