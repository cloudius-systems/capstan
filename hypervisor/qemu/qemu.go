/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"bufio"
	"fmt"
	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type VMConfig struct {
	Name        string // general
	Verbose     bool
	Cmd         string
	DisableKvm  bool
	Persist     bool
	InstanceDir string
	Monitor     string
	ConfigFile  string
	AioType     string
	Image       string // storage
	BackingFile bool
	Volumes     []string
	Memory      int64 // resources
	Cpus        int
	Networking  string // networking
	Bridge      string
	NatRules    []nat.Rule
	MAC         string
	VNCFile     string // VNC domain socket path
	KernelMode  bool
	KernelPath  string
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func DeleteVM(name string) error {
	dir := filepath.Join(util.ConfigDir(), "instances/qemu", name)
	c := &VMConfig{
		InstanceDir: dir,
		Monitor:     filepath.Join(dir, "osv.monitor"),
		Image:       filepath.Join(dir, "disk.qcow2"),
		ConfigFile:  filepath.Join(dir, "osv.config"),
		VNCFile:     filepath.Join(dir, "vnc-domain-socket"),
	}
	cmd := exec.Command("rm", "-f", c.Image, " ", c.Monitor, " ", c.ConfigFile, " ", c.VNCFile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("rm failed: %s, %s", c.Image, c.Monitor)
		return err
	}

	cmd = exec.Command("rmdir", c.InstanceDir)
	_, err = cmd.Output()
	if err != nil {
		fmt.Printf("rmdir failed: %s", c.InstanceDir)
		return err
	}

	return nil
}

func StopVM(name string) error {
	dir := filepath.Join(util.ConfigDir(), "instances/qemu", name)
	c := &VMConfig{
		Monitor: filepath.Join(dir, "osv.monitor"),
	}
	conn, err := net.Dial("unix", c.Monitor)
	if err != nil {
		// The instance is stopped already
		return nil
	}

	writer := bufio.NewWriter(conn)

	cmd := `{ "execute": "qmp_capabilities"}`
	writer.WriteString(cmd)

	cmd = `{ "execute": "system_powerdown" }`
	writer.WriteString(cmd)

	cmd = `{ "execute": "quit" }`
	writer.WriteString(cmd)

	writer.Flush()

	return nil
}

func GetVMStatus(name, dir string) (string, error) {
	c := &VMConfig{
		Monitor: filepath.Join(dir, "osv.monitor"),
	}
	_, err := net.Dial("unix", c.Monitor)
	if err != nil {
		return "Stopped", nil
	}

	return "Running", nil
}

func LoadConfig(name string) (*VMConfig, error) {
	dir := filepath.Join(util.ConfigDir(), "instances/qemu", name)
	file := filepath.Join(dir, "osv.config")
	c := VMConfig{}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Failed to open: %s\n", file)
		return nil, err
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func StoreConfig(c *VMConfig) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.ConfigFile, data, 0644)
}

func VMCommand(c *VMConfig, verbose bool, extra ...string) (*exec.Cmd, error) {
	if c.BackingFile {
		dir := c.InstanceDir
		err := os.MkdirAll(dir, 0775)
		if err != nil {
			fmt.Printf("mkdir failed: %s", dir)
			return nil, err
		}

		image, err := filepath.Abs(c.Image)
		if err != nil {
			fmt.Printf("Failed to open image %s\n", c.Image)
			return nil, err
		}
		backingFile := "backing_file=" + image
		newDisk := dir + "/disk.qcow2"

		if _, err := os.Stat(newDisk); os.IsNotExist(err) {
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-o", backingFile, newDisk)
			if stdout, err := cmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("qemu-img failed: %s\n%s", stdout, err)
			}
		}
		c.Image = newDisk
	}

	c.VNCFile = filepath.Join(c.InstanceDir, "vnc-domain-socket")

	if c.Cmd != "" {
		fmt.Printf("Setting cmdline: %s\n", c.Cmd)
		util.SetCmdLine(c.Image, c.Cmd)
	}

	if c.Persist {
		StoreConfig(c)
	}

	version, err := ProbeVersion()
	if err != nil {
		return nil, err
	}
	vmArgs, err := c.vmArguments(version)
	if err != nil {
		return nil, err
	}
	args := append(vmArgs, extra...)
	path, err := qemuExecutable()
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("Invoking QEMU at: %s with arguments:", path)
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
	return cmd, nil
}

func LaunchVM(c *VMConfig, verbose bool, extra ...string) (*exec.Cmd, error) {
	cmd, err := VMCommand(c, verbose, extra...)
	if err != nil {
		return nil, err
	}
	if c.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func ProbeVersion() (*Version, error) {
	path, err := qemuExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(path, "-version")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return ParseVersion(string(out))
}

func ParseVersion(text string) (*Version, error) {
	r, err := regexp.Compile("QEMU.*emulator version (\\d+)\\.(\\d+)(\\.)?(\\d?)?")
	if err != nil {
		return nil, err
	}
	version := r.FindStringSubmatch(text)
	if len(version) < 5 {
		return nil, fmt.Errorf("unable to parse QEMU version from '%s'", text)
	}
	major, err := strconv.Atoi(version[1])
	if err != nil {
		return nil, err
	}
	minor, err := strconv.Atoi(version[2])
	if err != nil {
		return nil, err
	}
	patch := 0
	if version[4] != "" {
		patch, err = strconv.Atoi(version[4])
		if err != nil {
			return nil, err
		}
	}
	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func (c *VMConfig) vmDriveCache() string {
	if util.IsDirectIOSupported(c.Image) {
		return "none"
	}
	return "unsafe"
}

func (c *VMConfig) ValidateVmArguments(version *Version) error {
	if c.AioType != "native" && c.AioType != "threads" {
		return fmt.Errorf("aio type must be [native|threads], got: %s", c.AioType)
	}

	return nil
}

func (c *VMConfig) vmArguments(version *Version) ([]string, error) {
	if err := c.ValidateVmArguments(version); err != nil {
		return []string{}, fmt.Errorf("argument validation failed: %s", err.Error())
	}

	args := make([]string, 0)
	args = append(args, "-vnc", "unix:"+c.VNCFile)
	args = append(args, "-m", strconv.FormatInt(c.Memory, 10))
	args = append(args, "-smp", strconv.Itoa(c.Cpus))
	if c.KernelMode {
		args = append(args, "-device", "virtio-blk-pci,id=blk0,drive=hd0")
	} else {
		args = append(args, "-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0")
	}
	args = append(args, "-drive", "file="+c.Image+",if=none,id=hd0,aio="+c.AioType+",cache="+c.vmDriveCache())
	if version.Major >= 1 && version.Minor >= 3 {
		args = append(args, "-device", "virtio-rng-pci")
	}
	args = append(args, "-chardev", "stdio,mux=on,id=stdio,signal=off")
	args = append(args, "-device", "isa-serial,chardev=stdio")
	if c.KernelMode {
		args = append(args, "-append", c.Cmd)
		args = append(args, "-kernel", c.KernelPath)
	}

	if volumes, err := hypervisor.ParseVolumes(c.Volumes); err == nil {
		for idx, v := range volumes {
			bootIndex := idx + 1
			driveId := fmt.Sprintf("hd%d", bootIndex)
			deviceId := fmt.Sprintf("blk%d", bootIndex)
			args = append(
				args,
				"-drive",
				fmt.Sprintf("file=%s,if=none,id=%s,aio=%s,cache=%s,format=%s", v.Path, driveId, v.AioType, v.Cache, v.Format),
			)
			args = append(
				args,
				"-device",
				fmt.Sprintf("virtio-blk-pci,id=%s,bootindex=%d,drive=%s", deviceId, bootIndex, driveId),
			)
		}
	} else {
		return nil, err
	}

	net, err := c.vmNetworking()
	if err != nil {
		return nil, err
	}
	args = append(args, net...)
	monitor := fmt.Sprintf("socket,id=charmonitor,path=%s,server=on,wait=off", c.Monitor)
	args = append(args, "-chardev", monitor, "-mon", "chardev=charmonitor,id=monitor,mode=control")
	if !c.DisableKvm && runtime.GOOS == "linux" && checkKVM() {
		args = append(args, "-enable-kvm", "-cpu", "host,+x2apic")
	}
	if runtime.GOOS == "darwin" {
		if checkHAXM() {
			args = append(args, "-accel", "hax")
		} else {
			fmt.Println("Running QEMU without acceleration: please install Intel HAXM from https://github.com/intel/haxm/releases")
		}
	}
	return args, nil
}

func (c *VMConfig) vmMAC() (net.HardwareAddr, error) {
	if c.MAC != "" {
		return net.ParseMAC(c.MAC)
	}
	return util.GenerateMAC()
}

func (c *VMConfig) vmNetworking() ([]string, error) {
	args := make([]string, 0)
	switch c.Networking {
	case "bridge":
		mac, err := c.vmMAC()
		if err != nil {
			return nil, err
		}

		bridgeHelper, err := qemuBridgeHelper()
		if err != nil {
			return nil, err
		}

		args = append(args, "-netdev", fmt.Sprintf("bridge,id=hn0,br=%s,helper=%s", c.Bridge, bridgeHelper), "-device", fmt.Sprintf("virtio-net-pci,netdev=hn0,id=nic1,mac=%s", mac.String()))
		return args, nil
	case "nat":
		netdevValue := "user,id=un0,net=192.168.122.0/24,host=192.168.122.1"
		for _, portForward := range c.NatRules {
			netdevValue = netdevValue + fmt.Sprintf(",hostfwd=tcp::%s-:%s", portForward.HostPort, portForward.GuestPort)
		}
		args = append(args, "-netdev", netdevValue, "-device", "virtio-net-pci,netdev=un0")
		return args, nil
	case "tap":
		mac, err := c.vmMAC()
		if err != nil {
			return nil, err
		}
		args = append(args, "-netdev", fmt.Sprintf("tap,id=hn0,ifname=%s,script=no,downscript=no", c.Bridge), "-device", fmt.Sprintf("virtio-net-pci,netdev=hn0,id=nic1,mac=%s", mac.String()))
		return args, nil
	case "vhost":
		mac, err := c.vmMAC()
		if err != nil {
			return nil, err
		}
		args = append(args, "-net", fmt.Sprintf("nic,model=virtio,macaddr=%s,netdev=nic-0", mac.String()), "-netdev", "tap,id=nic-0,vhost=on")
		return args, nil
	}

	return nil, fmt.Errorf("%s: networking not supported", c.Networking)
}

func qemuExecutable() (string, error) {
	paths := []string{
		"/usr/bin/qemu-system-x86_64",
		"/usr/libexec/qemu-kvm",
		"/usr/local/bin/qemu-system-x86_64",
	}
	path := os.Getenv("CAPSTAN_QEMU_PATH")
	if len(path) > 0 {
		paths = append([]string{path}, paths...)
	}
	for _, path = range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("No QEMU installation found. Use the CAPSTAN_QEMU_PATH environment variable to specify its path.")
}

func qemuBridgeHelper() (string, error) {
	paths := []string{
		"/usr/libexec",
		"/usr/lib/qemu",
		"/usr/lib",
	}

	// Use ENV variable if it exists. This allows users to set the location if not avaliable
	// in standard directories.
	bridgeHelper := os.Getenv("CAPSTAN_QEMU_BRIDGE_HELPER")
	if bridgeHelper != "" {
		if _, err := os.Stat(bridgeHelper); err == nil {
			return bridgeHelper, nil
		}
	}

	// If the ENV setting was not available or the file does not exist, try standard locations
	for _, path := range paths {
		bridgeHelper := filepath.Join(path, "qemu-bridge-helper")
		if _, err := os.Stat(bridgeHelper); err == nil {
			return bridgeHelper, nil
		}
	}

	return "", fmt.Errorf("No QEMU bridge helper (qemu-bridge-helper) found. Use CAPSTAN_QEMU_BRIDGE_HELPER to set the path to qemu-bridge-helper.")
}

func checkKVM() bool {
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0666)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

func checkHAXM() bool {
	cmd := exec.Command("kextstat", "-l", "-b", "com.intel.kext.intelhaxm")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "com.intel.kext.intelhaxm")
}

func CreateVolume(path, format string, sizeMB int64) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("Volume already exists")
	}
	cmd := exec.Command("qemu-img", "create", "-f", format, path, fmt.Sprintf("%dM", sizeMB))
	if stdout, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s\n%s", stdout, err)
	}
	return nil
}
