package richcontainer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runc/utils"
)

const (
	systemdLauncherName       = "systemd"
	systemdBinRootfsPath      = "/usr/lib/systemd/systemd"
	systemdDefaultDescription = "rich container run mode"
	systemdServiceFilePath    = "/etc/systemd/system/richcontainer.service"
)

func init() {
	RegisterLauncher(&systemdLauncher{})
}

type systemdLauncher struct {
}

func (l *systemdLauncher) Name() string {
	return systemdLauncherName
}

//todo:
func (l *systemdLauncher) Launch(opt *prehook.HookOptions, spec *specs.Spec) error {
	rootfs := opt.RootfsDir

	//check has systemd
	_, err := os.Stat(filepath.Join(rootfs, systemdBinRootfsPath))
	if err != nil {
		return err
	}

	cmd := spec.Process.Args

	if len(cmd) == 0 {
		return errors.New("no cmd set in process of container runtime spec")
	}

	//find PATH in env
	sysPaths := []string{}

	for _, env := range spec.Process.Env {
		kvs := strings.Split(env, "=")
		if len(kvs) == 2 && kvs[0] == "PATH" {
			sysPaths = strings.Split(kvs[1], ":")
			break
		}
	}

	path := cmd[0]
	abPath, err := utils.FindAbPathInRootfs(path, rootfs, sysPaths)
	if err != nil {
		return err
	}

	cmd[0] = abPath

	config := &systemdConfig{
		unit: &systemdUnitConfig{
			Description: systemdDefaultDescription,
		},
		service: &systemdServiceConfig{
			Type:      "simple",
			ExecStart: strings.Join(cmd, " "),
		},
		install: &systemdInstallConfig{
			WantedBy: "multi-user.target",
		},
	}

	err = l.writeServiceFile(config, filepath.Join(rootfs, systemdServiceFilePath))
	if err != nil {
		return err
	}

	//link service to multi-user dir
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	defer os.Chdir(currentDir)

	err = os.Chdir(filepath.Join(rootfs, "/etc/systemd/system/multi-user.target.wants"))
	if err != nil {
		return err
	}

	err = l.linkService("../richcontainer.service", "richcontainer.service")
	if err != nil {
		return err
	}

	newCmd := []string{systemdBinRootfsPath}
	spec.Process.Args = newCmd

	return nil
}

type systemdUnitConfig struct {
	Description string
}

type systemdServiceConfig struct {
	//type :simple
	Type      string
	ExecStart string
}

type systemdInstallConfig struct {
	//default multi-user.target
	WantedBy string
}

type systemdConfig struct {
	unit    *systemdUnitConfig
	service *systemdServiceConfig
	install *systemdInstallConfig
}

func (l *systemdLauncher) isSetRichContainerService(rootfs string) (bool, error) {
	_, err := os.Stat(filepath.Join(rootfs, systemdServiceFilePath))
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	}

	data, err := ioutil.ReadFile(filepath.Join(rootfs, systemdServiceFilePath))
	if err != nil {
		return false, err
	}

	has := strings.Contains(string(data), fmt.Sprintf("Description=%s", systemdDefaultDescription))
	if has {
		return true, nil
	}

	return false, nil
}

func (l *systemdLauncher) writeServiceFile(config *systemdConfig, filePath string) error {
	dir := filepath.Dir(filePath)

	err := os.MkdirAll(dir, 0x755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0x755)
	if err != nil {
		return err
	}

	defer f.Close()

	//write config
	if config.unit != nil {
		f.WriteString("[Unit]\n")
		f.WriteString(fmt.Sprintf("Description=%s\n", systemdDefaultDescription))
		f.WriteString("\n")
	}

	if config.service != nil {
		f.WriteString("[Service]\n")

		if config.service.Type != "" {
			f.WriteString(fmt.Sprintf("Type=%s\n", config.service.Type))
		}

		if config.service.ExecStart != "" {
			f.WriteString(fmt.Sprintf("ExecStart=%s\n", config.service.ExecStart))
		}

		f.WriteString("\n")
	}

	if config.install != nil {
		f.WriteString("[Install]\n")

		if config.install.WantedBy != "" {
			f.WriteString(fmt.Sprintf("WantedBy=%s\n", config.install.WantedBy))
		}
		f.WriteString("\n")
	}

	return nil
}

func (l *systemdLauncher) linkService(serviceFilePath string, target string) error {
	_, err := os.Stat(target)
	if err == nil {
		//if exists; remove it
		e := os.Remove(target)
		if e != nil {
			return e
		}
	}

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return exec.Command("ln", "-s", serviceFilePath, target).Run()
}
