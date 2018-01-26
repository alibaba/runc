package prehook

//hook in pre start container

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"sync"
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