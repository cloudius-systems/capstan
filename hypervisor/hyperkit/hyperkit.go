package hyperkit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type VMConfig struct {
	Name        string
	Dir         string
	Image       string
	VmlinuzPath string
	Cmd         string
	Memory      int64
	Cpus        int
	Networking  string
	Bridge      string
	InstanceDir string
	ConfigFile  string
	MAC         string
}

const (
	legacyVPNKitSock  = "Library/Containers/com.docker.docker/Data/s50"
	defaultVPNKitSock = "Library/Containers/com.docker.docker/Data/vpnkit.eth.sock"
)

func LaunchVM(c *VMConfig, verbose bool, extra ...string) (*exec.Cmd, error) {

	vmArgs, err := c.vmArguments()
	if err != nil {
		return nil, err
	}
	args := append(vmArgs, extra...)
	path, err := hyperkitExecutable()
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("Invoking HYPERKIT at: %s with arguments:", path)
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				fmt.Printf("\n  %s", arg)
			} else {
				fmt.Printf(" %s", arg)
			}
		}
		fmt.Printf("\n")
	}

	cmd := exec.Command(path, args...)

	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func (c *VMConfig) vmArguments() ([]string, error) {
	args := make([]string, 0)
	args = append(args, "-A")                                                      // Enable ACPI
	args = append(args, "-x")                                                      // Enable x2APIC
	args = append(args, "-c", strconv.Itoa(c.Cpus))                                // Number of cpus
	args = append(args, "-m", strconv.FormatInt(c.Memory, 10)+"M")           // Memory
	args = append(args, "-f", fmt.Sprintf("kexec,%s,,%s", c.VmlinuzPath, c.Cmd)) //firmware, kernel and commandline
	args = append(args, "-l", "com1,stdio")                                        // ???
	args = append(args, "-s", "0:0,hostbridge")                                    // PCI bus
	args = append(args, "-s", "31,lpc")                                            // ???

	nextSlot := 1
	args = append(args, "-s", fmt.Sprintf("%d:0,virtio-blk,%s", nextSlot, c.Image)) // VirtIO block device
	nextSlot++

	switch c.Networking {
	case "vpnkit":
		vpnSockPath, err := vpnSocketPath("auto")
		if err != nil {
			return nil, err
		}
		args = append(args, "-s", fmt.Sprintf("%d:0,virtio-vpnkit,path=%s", nextSlot, vpnSockPath))
		nextSlot++
		vsockDirPath := filepath.Join(c.InstanceDir, "vsockState")
		err = os.MkdirAll(vsockDirPath, 0775)
		if err != nil {
			return args, errors.New(fmt.Sprintf("%s: failed to create vstate dir", vsockDirPath))
		}
		//TODO: Potentially add ',guest_forwards=8080' to the string below to handle port forwarding
		args = append(args, "-s", fmt.Sprintf("%d,virtio-sock,guest_cid=%d,path=%s",
			nextSlot, 3, vsockDirPath))
		nextSlot++
	case "vnet":
		args = append(args, "-s", fmt.Sprintf("%d:0,virtio-net", nextSlot))
		nextSlot++
	}

	return args, nil
}

func vpnSocketPath(vpnkitsock string) (string, error) {
	if vpnkitsock == "auto" {
		vpnkitsock = filepath.Join(getHome(), defaultVPNKitSock)
		if _, err := os.Stat(vpnkitsock); err != nil {
			vpnkitsock = filepath.Join(getHome(), legacyVPNKitSock)
		}
	}
	if vpnkitsock == "" {
		return "", nil
	}

	vpnkitsock = filepath.Clean(vpnkitsock)
	_, err := os.Stat(vpnkitsock)
	if err != nil {
		return "", err
	}
	return vpnkitsock, nil
}

func getHome() string {
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return os.Getenv("HOME")
}

var defaultHyperKits = []string{"hyperkit",
	"com.docker.hyperkit",
	"/usr/local/bin/hyperkit",
	"/Applications/Docker.app/Contents/Resources/bin/hyperkit",
	"/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit"}

func hyperkitExecutable() (string, error) {
	paths := defaultHyperKits
	path := os.Getenv("CAPSTAN_HYPERKIT_PATH")
	if len(path) > 0 {
		paths = append([]string{path}, paths...)
	}
	for _, path = range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("No HYPERKIT installation found. Use the CAPSTAN_HYPERKIT_PATH environment variable to specify its path.")
}
