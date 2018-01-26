package richcontainer

import (
	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runtime-spec/specs-go"
)


const(
	systemdLauncherName = "systemd"
)

func init()  {
	RegisterLauncher(&systemdLauncher{})
}

type systemdLauncher struct {

}

func (l *systemdLauncher) Name() string {
	return systemdLauncherName
}

//todo:
func (l *systemdLauncher) Launch(opt *prehook.HookOptions, spec *specs.Spec) error {
	return nil
}

