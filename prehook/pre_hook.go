package prehook

//hook in pre create container

import (
	"sync"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

type HookOptions struct {
	//container rootfs path
	RootfsDir string
	//container id
	ID        string
}

type HookFunc func(opt *HookOptions, spec *specs.Spec) error

type HookRegistration struct {
	Type		string
	RunFunc		HookFunc
}

func RegisterPreHook(f *HookRegistration)  {
	registerMutex.Lock()
	defer registerMutex.Unlock()

	if f.Type == ""{
		panic("not set hook type")
	}

	hookRegistrations = append(hookRegistrations, f)
}

var(
	hookRegistrations = []*HookRegistration{}
	registerMutex = &sync.Mutex{}
)

func PreHook(opt *HookOptions, spec *specs.Spec) error {
	for _,hook := range hookRegistrations{
		err := hook.RunFunc(opt, spec)
		if err != nil{
			return err
		}
	}

	return nil
}

func CreateHookOptions(context* cli.Context, spec *specs.Spec) (*HookOptions,error) {
	rootfsPath := ""

	if filepath.IsAbs(spec.Root.Path){
		rootfsPath = spec.Root.Path
	}else {
		p,err := filepath.Abs(filepath.Join(context.String("bundle"), spec.Root.Path))
		if err != nil {
			return nil, err
		}

		rootfsPath = p
	}

	return &HookOptions{
		RootfsDir: rootfsPath,
		ID: context.Args().First(),
	},nil
}