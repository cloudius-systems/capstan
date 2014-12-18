/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"bufio"
	"fmt"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

type VMConfig struct {
	Name        string
	Image       string
	Verbose     bool
	Memory      int64
	Cpus        int
	Networking  string
	Bridge      string
	NatRules    []nat.Rule
	BackingFile bool
	InstanceDir string
	Monitor     string
	ConfigFile  string
	MAC         string
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func DeleteVM(name string) error {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
	c := &VMConfig{
		InstanceDir: dir,
		Monitor:     filepath.Join(dir, "osv.monitor"),
		Image:       filepath.Join(dir, "disk.qcow2"),
		ConfigFile:  filepath.Join(dir, "osv.config"),
	}
	cmd := exec.Command("rm", "-f", c.Image, " ", c.Monitor, " ", c.ConfigFile)
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
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
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
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
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

func VMCommand(c *VMConfig, extra ...string) (*exec.Cmd, error) {
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
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-o", backingFile, newDisk)
			_, err = cmd.Output()
			if err != nil {
				fmt.Printf("qemu-img failed: %s", newDisk)
				return nil, err
			}
		}
		c.Image = newDisk
	}

	StoreConfig(c)

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
	cmd := exec.Command(path, args...)
	return cmd, nil
}

func LaunchVM(c *VMConfig, extra ...string) (*exec.Cmd, error) {
	cmd, err := VMCommand(c, extra...)
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

func (c *VMConfig) vmArguments(version *Version) ([]string, error) {
	args := make([]string, 0)
	args = append(args, "-nographic")
	args = append(args, "-m", strconv.FormatInt(c.Memory, 10))
	args = append(args, "-smp", strconv.Itoa(c.Cpus))
	args = append(args, "-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0")
	args = append(args, "-drive", "file=" + c.Image + ",if=none,id=hd0,aio=native,cache=" + c.vmDriveCache())
	if version.Major >= 1 && version.Minor >= 3 {
		args = append(args, "-device", "virtio-rng-pci")
	}
	args = append(args, "-chardev", "stdio,mux=on,id=stdio,signal=off")
	args = append(args, "-device", "isa-serial,chardev=stdio")
	net, err := c.vmNetworking()
	if err != nil {
		return nil, err
	}
	args = append(args, net...)
	monitor := fmt.Sprintf("socket,id=charmonitor,path=%s,server,nowait", c.Monitor)
	args = append(args, "-chardev", monitor, "-mon", "chardev=charmonitor,id=monitor,mode=control")
	if runtime.GOOS == "linux" {
		args = append(args, "-enable-kvm", "-cpu", "host,+x2apic")
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
		args = append(args, "-netdev", fmt.Sprintf("bridge,id=hn0,br=%s,helper=/usr/libexec/qemu-bridge-helper", c.Bridge), "-device", fmt.Sprintf("virtio-net-pci,netdev=hn0,id=nic1,mac=%s", mac.String()))
		return args, nil
	case "nat":
		args = append(args, "-netdev", "user,id=un0,net=192.168.122.0/24,host=192.168.122.1", "-device", "virtio-net-pci,netdev=un0")
		for _, portForward := range c.NatRules {
			redirect := fmt.Sprintf("tcp:%s::%s", portForward.HostPort, portForward.GuestPort)
			args = append(args, "-redir", redirect)
		}
		return args, nil
        case "tap":
		mac, err := c.vmMAC()
		if err != nil {
			return nil, err
		}
		args = append(args, "-netdev", fmt.Sprintf("tap,id=hn0,ifname=%s,script=no,downscript=no", c.Bridge), "-device", fmt.Sprintf("virtio-net-pci,netdev=hn0,id=nic1,mac=%s", mac.String()))
		return args, nil
	}
	return nil, fmt.Errorf("%s: networking not supported", c.Networking)
}

func qemuExecutable() (string, error) {
	paths := []string{
		"/usr/bin/qemu-system-x86_64",
		"/usr/libexec/qemu-kvm",
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
