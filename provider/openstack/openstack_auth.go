package openstack

import (
	"fmt"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/urfave/cli"
	"os"
)

// OPENSTACK_CREDENTIALS_FLAGS is a list of argumets that are used for OpenStack authentication.
// Append it to Flags list to enable direct authentication i.e. without environment variables.
var OPENSTACK_CREDENTIALS_FLAGS = []cli.Flag{
	cli.StringFlag{Name: "OS_AUTH_URL", Usage: "OpenStack auth url (e.g. http://10.0.2.15:5000/v2.0)"},
	cli.StringFlag{Name: "OS_TENANT_ID", Usage: "OpenStack tenant id (e.g. 3dfe7bf545ff4885a3912a92a4a5f8e0)"},
	cli.StringFlag{Name: "OS_TENANT_NAME", Usage: "OpenStack tenant name (e.g. admin)"},
	cli.StringFlag{Name: "OS_PROJECT_NAME", Usage: "OpenStack project name (e.g. admin)"},
	cli.StringFlag{Name: "OS_USERNAME", Usage: "OpenStack username (e.g. admin)"},
	cli.StringFlag{Name: "OS_PASSWORD", Usage: "OpenStack password (*TODO*: leave blank to be prompted)"},
	cli.StringFlag{Name: "OS_REGION_NAME", Usage: "OpenStack username (e.g. RegionOne)"},
}

// ObtainCredentials attempts to obtain OpenStack credentials either from command args eihter from env.
// If at least one command argument regarding credentials is non-empty, environment is ignored.
func ObtainCredentials(c *cli.Context, verbose bool) (*gophercloud.AuthOptions, error) {
	// Obtain credentials passed as script arguments
	credentials, err := AuthOptionsFromArgs(c)
	if err != nil {
		return nil, err
	}

	// Obtain credentials from environment if they were not passed as arguments
	if credentials == nil {
		if verbose {
			fmt.Println("Using OpenStack credentials from environment variables")
		}

		// Allocate.
		credentials = new(gophercloud.AuthOptions)

		// Retrieve credentials from environment variables.
		*credentials, err = openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, err
		}
	}
	return credentials, nil
}

// AuthOptionsFromArgs fetches OpenStack credentials from command-line arguments.
// If not even a single argument is passed, then return nil without error.
func AuthOptionsFromArgs(c *cli.Context) (*gophercloud.AuthOptions, error) {
	authURL := c.String("OS_AUTH_URL")
	username := c.String("OS_USERNAME")
	userID := c.String("OS_USERID")
	password := c.String("OS_PASSWORD")
	tenantID := c.String("OS_TENANT_ID")
	tenantName := c.String("OS_TENANT_NAME")
	domainID := c.String("OS_DOMAIN_ID")
	domainName := c.String("OS_DOMAIN_NAME")

	// If non of the arguments is set, user does not make use of comand line authentication.
	if authURL == "" &&
		username == "" &&
		userID == "" &&
		password == "" &&
		tenantID == "" &&
		tenantName == "" &&
		domainID == "" &&
		domainName == "" {
		return nil, nil
	}

	if authURL == "" {
		return nil, fmt.Errorf("Argument --OS_AUTH_URL needs to be set.")
	}

	if username == "" && userID == "" {
		return nil, fmt.Errorf("Argument --OS_USERNAME needs to be set.")
	}

	if password == "" {
		return nil, fmt.Errorf("Argument --OS_PASSWORD needs to be set.")
	}

	if tenantName == "" && tenantID == "" {
		return nil, fmt.Errorf("Argument --OS_TENANT_NAME needs to be set.")
	}

	ao := gophercloud.AuthOptions{
		IdentityEndpoint: authURL,
		UserID:           userID,
		Username:         username,
		Password:         password,
		TenantID:         tenantID,
		TenantName:       tenantName,
		DomainID:         domainID,
		DomainName:       domainName,
	}

	return &ao, nil
}

// GetClients authenticates against OpenStack Identity and obtains Nova and Glance client.
// Pass nil credentials to fetch it from environment.
func GetClients(credentials *gophercloud.AuthOptions, verbose bool) (*gophercloud.ServiceClient, *gophercloud.ServiceClient, error) {
	// Perform authentication.
	provider, err := openstack.AuthenticatedClient(*credentials)
	if err != nil {
		return nil, nil, err
	}

	// Obtain different clients
	clientNova, _ := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	clientGlance, _ := openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	return clientNova, clientGlance, nil
}
