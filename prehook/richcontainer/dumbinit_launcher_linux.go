package richcontainer

import (
	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runtime-spec/specs-go"
	"os/exec"
	"os"
	"path/filepath"
	"io"
	"errors"
	"fmt"
)

//use dumb-init as init process in container if not set launcher

const(
	dumbInitLauncherName = "dumbinit"
	dumbInitAppName = "dumb-init"

	dumbInitRootfsPath = "/usr/bin/dumb-init"
)

func init()  {
	RegisterLauncher(&dumbInitLauncher{})
}

type dumbInitLauncher struct {
}

func (l *dumbInitLauncher) Name() string {
	return dumbInitLauncherName
}

//todo:
func (l *dumbInitLauncher) Launch(opt *prehook.HookOptions, spec *specs.Spec) error {
	fmt.Println("rich container mode in dumb-init mode")

	//find dumb-init path in node
	path,err := exec.LookPath(dumbInitAppName)
	if err != nil{
		return err
	}

	abPath,err := filepath.Abs(path)
	if err != nil{
		return err
	}

	err = l.copyToContainerRootfs(abPath, opt.RootfsDir)
	if err != nil{
		return err
	}

	//entrypoint
	args := spec.Process.Args
	if args == nil || len(args) == 0{
		return errors.New("not set args")
	}

	newArgs := []string{dumbInitRootfsPath, "--"}
	newArgs = append(newArgs, args...)

	spec.Process.Args = newArgs
	//set user to admin
	spec.Process.User.Username = "admin"

	return nil
}

func (l *dumbInitLauncher) copyToContainerRootfs(binPath string, rootfs string) error {
	rootfsBinPath := filepath.Join(rootfs, dumbInitRootfsPath)
	_,err := os.Stat(rootfsBinPath)

	if err == nil {
		return nil
	}

	if !os.IsNotExist(err){
		return err
	}

	//mkdir /usr/bin
	err = os.MkdirAll(filepath.Dir(rootfsBinPath), 0x0755)
	if err != nil{
		return err
	}

	fin, err := os.Open(binPath)
	if err != nil{
		return err
	}

	defer fin.Close()

	fout,err := os.OpenFile(rootfsBinPath, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0x0755)
	if err != nil{
		return err
	}

	defer fout.Close()

	_,err = io.Copy(fout, fin)
	if err != nil{
		return err
	}

	return nil
}

