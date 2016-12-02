package cmd

import (
	"fmt"
	"github.com/mikelangelo-project/capstan/provider/openstack"
	"github.com/mikelangelo-project/capstan/util"
	"github.com/urfave/cli"
	"os"
)

// OpenStackPush picks best flavor, composes package, builds .qcow2 image and uploads it to OpenStack.
func OpenStackPush(c *cli.Context) error {
	verbose := c.Bool("verbose")
	imageName := c.Args().First()
	pullMissing := c.Bool("pull-missing")

	if imageName == "" {
		return fmt.Errorf("USAGE: capstan stack push [command options] image-name")
	}

	// Use the provided repository.
	repo := util.NewRepo(c.GlobalString("u"))

	// Get temporary name of the application.
	appName := imageName
	if verbose {
		fmt.Printf("appName: %s\n", appName)
	}

	// Authenticate against OpenStack Identity
	credentials, err := openstack.ObtainCredentials(c, verbose)
	if err != nil {
		return err
	}
	clientNova, clientGlance, err := openstack.GetClients(credentials, verbose)
	if err != nil {
		return err
	}

	// Pick appropriate flavor.
	diskMB, _ := util.ParseMemSize(c.String("size"))
	flavor, err := openstack.GetOrPickFlavor(clientNova, c.String("flavor"), diskMB, -1, verbose)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Picked flavor %s (disk %dGB, memory %dMB)\n", flavor.Name, flavor.Disk, flavor.RAM)
	}

	// Always use the current directory for the package to compose.
	packageDir, _ := os.Getwd()

	// flavor.Disk is in GB, we need MB
	sizeMB := 1024 * int64(flavor.Disk)

	// Compose image locally.
	fmt.Printf("Creating image of user-usable size %d MB.\n", sizeMB)
	err = ComposePackage(repo, sizeMB, false, verbose, pullMissing, c.String("boot"), packageDir, appName, c.String("run"))
	if err != nil {
		return err
	}

	// Remove local copy of image after uploaded to OpenStack.
	if !c.Bool("keep-image") {
		defer repo.RemoveImage(appName)
	} else {
		if verbose {
			fmt.Println("Keeping image locally. Please remove it manually.")
		}
	}

	// Push to OpenStack.
	fmt.Println("Uploading image to OpenStack. This may take a while.")
	imageFilepath := repo.ImagePath("qemu", appName)
	openstack.PushImage(clientGlance, imageName, imageFilepath, flavor, verbose)
	fmt.Printf("Image '%s' [src: %s] successfully uploaded to OpenStack.\n", imageName, packageDir)

	return nil
}

// OpenStackRun picks best flavor for image and runs instacne(s) on OpenStack.
func OpenStackRun(c *cli.Context) error {
	verbose := c.Bool("verbose")
	imageName := c.Args().First()
	count := c.Int("count")

	if imageName == "" {
		fmt.Println("USAGE: capstan stack run [command options] image-name")
		os.Exit(1)
	}

	name := c.String("name")
	if name == "" {
		name = fmt.Sprintf("instance-of-%s", imageName)
	}

	// Authenticate against OpenStack Identity
	credentials, err := openstack.ObtainCredentials(c, verbose)
	if err != nil {
		return err
	}
	clientNova, _, err := openstack.GetClients(credentials, verbose)
	if err != nil {
		return err
	}

	// Obtain image metadata.
	image, err := openstack.GetImageMeta(clientNova, imageName, verbose)
	if err != nil {
		return err
	}

	// Pick appropriate flavor.
	diskMB := 1024 * int64(image.MinDisk)
	memoryMB, _ := util.ParseMemSize(c.String("mem"))
	flavor, err := openstack.GetOrPickFlavor(clientNova, c.String("flavor"), diskMB, memoryMB, verbose)
	if err != nil {
		return err
	}
	fmt.Printf("Picked flavor %s (disk %dGB, memory %dMB)\n", flavor.Name, flavor.Disk, flavor.RAM)

	// Make sure that flavor meets minimum requirements for the image
	if image.MinDisk > flavor.Disk || image.MinRAM > flavor.RAM {
		fmt.Printf("ABORTED: Flavor '%s' (disk %dGB, memory %dMB) violates image minimum requirements (disk >= %dGB, memory >= %dMB)\n",
			flavor.Name, flavor.Disk, flavor.RAM, image.MinDisk, image.MinRAM)
		os.Exit(1)
	}

	// Launch instances.
	fmt.Printf("Launching %d instances from image '%s'...\n", count, imageName)
	err = openstack.LaunchInstances(clientNova, name, imageName, flavor.Name, count, verbose)
	if err != nil {
		return err
	}
	fmt.Println("Instances launched.")

	return nil
}
