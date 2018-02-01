package richcontainer

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runc/prehook"
	"strings"
	"fmt"
)

func init()  {
	prehook.RegisterPreHook(&prehook.HookRegistration{
		Type: "richContainer",
		RunFunc: func(opt *prehook.HookOptions, spec *specs.Spec) error {
			if !isRichMode(spec){
				return nil
			}
			launcher,err := getRichModeLauncher(spec)
			if err != nil{
				return err
			}

			return launcher.Launch(opt, spec)
		},
	})
}

type RichContainerLauncher interface {
	Name()						      string
	Launch(opt *prehook.HookOptions, spec *specs.Spec)    error
}

type richLauncherManager struct {
	launchMapper		map[string]RichContainerLauncher
	defaultLauncher		string
}

var(
	launcherManager = &richLauncherManager{
		launchMapper:  map[string]RichContainerLauncher{},
		//set default launch to dumpinit
		defaultLauncher: dumbInitLauncherName,
	}
)

func RegisterLauncher(launcher RichContainerLauncher) error {
	_,ok := launcherManager.launchMapper[launcher.Name()]
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

const(
//	rich_mode_env = "ali_run_mode=common_vm"
	rich_mode_env = "rich_mode=true"
	rich_mode_launch_env = "rich_mode_launch_manner"
	rich_mode_script = "initscript"
)

func isRichMode(spec *specs.Spec) bool {
	envs := spec.Process.Env
	for _,env := range envs{
		if strings.TrimSpace(env) == rich_mode_env{
			return true
		}
	}

	return false
}

func getRichModeLauncher(spec *specs.Spec) (RichContainerLauncher, error) {
	var launcherName string = ""
	var launcher RichContainerLauncher = nil

	envs := spec.Process.Env

	for _,env := range envs{
		kvs := strings.Split(env, "=")
		if len(kvs) == 2 {
			if kvs[0] == rich_mode_launch_env {
				launcherName = kvs[1]
				break
			}
		}
	}

	if launcherName == ""{
		launcher = GetDefaultLauncher()
	}else {
		launcher = GetLauncher(launcherName)
	}

	if launcher == nil{
		return nil, fmt.Errorf("not found rich container launcher %s", launcherName)
	}

	return launcher, nil
}


