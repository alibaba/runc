package richcontainer

import (
	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runtime-spec/specs-go"
)


const(
	initdLauncherName = "init"
)

func init()  {
	RegisterLauncher(&initdLauncher{})
}

type initdLauncher struct {

}

func (l *initdLauncher) Name() string {
	return initdLauncherName
}

//todo:
func (l *initdLauncher) Launch(opt *prehook.HookOptions, spec *specs.Spec) error {
	return nil
}

