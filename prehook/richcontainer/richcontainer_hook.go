package richcontainer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runc/utils"
)

var (
	log = utils.GetLogger()
)

func init() {
	prehook.RegisterPreHook(&prehook.HookRegistration{
		Type: "richContainer",
		RunFunc: func(opt *prehook.HookOptions, spec *specs.Spec) error {
			if !isRichMode(spec) {
				return nil
			}

			launcher, err := getRichModeLauncher(spec)
			if err != nil {
				return err
			}

			if err = commonHook(opt, spec); err != nil {
				return err
			}

			return launcher.Launch(opt, spec)
		},
	})
}

type RichContainerLauncher interface {
	Name() string
	Launch(opt *prehook.HookOptions, spec *specs.Spec) error
}

type richLauncherManager struct {
	launchMapper    map[string]RichContainerLauncher
	defaultLauncher string
}

var (
	launcherManager = &richLauncherManager{
		launchMapper: map[string]RichContainerLauncher{},
		//set default launch to dumpinit
		defaultLauncher: dumbInitLauncherName,
	}
)

func RegisterLauncher(launcher RichContainerLauncher) error {
	_, ok := launcherManager.launchMapper[launcher.Name()]
	if ok {
		return fmt.Errorf("launcher %s has been register", launcher.Name())
	}

	launcherManager.launchMapper[launcher.Name()] = launcher
	return nil
}

//return nil if not found launcher
func GetLauncher(name string) RichContainerLauncher {
	launcher, ok := launcherManager.launchMapper[name]
	if ok {
		return launcher
	}

	return nil
}

func GetDefaultLauncher() RichContainerLauncher {
	launcher, ok := launcherManager.launchMapper[launcherManager.defaultLauncher]
	if ok {
		return launcher
	}

	return nil
}

const (
	richModeEnv       = "rich_mode=true"
	richModeLaunchEnv = "rich_mode_launch_manner"
	rich_mode_script  = "initscript"
)

func isRichMode(spec *specs.Spec) bool {
	envs := spec.Process.Env
	for _, env := range envs {
		if strings.TrimSpace(env) == richModeEnv {
			return true
		}
	}

	return false
}

func getRichModeLauncher(spec *specs.Spec) (RichContainerLauncher, error) {
	var launcherName string = ""
	var launcher RichContainerLauncher = nil

	envs := spec.Process.Env

	for _, env := range envs {
		kvs := strings.Split(env, "=")
		if len(kvs) == 2 {
			if kvs[0] == richModeLaunchEnv {
				launcherName = kvs[1]
				break
			}
		}
	}

	if launcherName == "" {
		launcher = GetDefaultLauncher()
	} else {
		launcher = GetLauncher(launcherName)
	}

	if launcher == nil {
		return nil, fmt.Errorf("not found rich container launcher %s", launcherName)
	}

	return launcher, nil
}

const (
	persistentEnvShFile = "/etc/profile.d/pouchenv.sh"
	persistentEnvShDir  = "/etc/profile.d"
)

func commonHook(opt *prehook.HookOptions, spec *specs.Spec) error {
	//persistent env to /etc/profile.d/pouchenv.sh
	rootfs := opt.RootfsDir
	envMap := map[string]string{}

	for _, env := range spec.Process.Env {
		kvs := strings.Split(env, "=")
		if len(kvs) == 2 {
			envMap[kvs[0]] = kvs[1]
		}
	}

	//mkdir $roofs/etc/profile.d/
	err := os.MkdirAll(filepath.Join(rootfs, persistentEnvShDir), 0x755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(rootfs, persistentEnvShFile), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0x755)
	if err != nil {
		return err
	}

	defer f.Close()

	for k, v := range envMap {
		f.WriteString(fmt.Sprintf("export %s=\"%s\"\n", k, v))
	}

	return nil
}
