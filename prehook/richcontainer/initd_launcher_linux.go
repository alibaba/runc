package richcontainer

import (
	"github.com/opencontainers/runc/prehook"
	"github.com/opencontainers/runtime-spec/specs-go"
	"path/filepath"
	"os"
	"strings"
	"fmt"
	"errors"
	"io/ioutil"
	"os/exec"
)


const(
	initdLauncherName = "sbin-init"
	initBinPath = "/sbin/init"

	defaultInitScriptPath = "/etc/rc.d/init.d/richContainer"
	defaultInitScriptName = "richContainer"
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
	rootfs := opt.RootfsDir

	//check has /sbin/init
	_,err := os.Stat(filepath.Join(rootfs, initBinPath))
	if err != nil{
		return err
	}

	cmd := spec.Process.Args
	config := &initScriptConfig{
		startCmd: strings.Join(cmd, " "),
	}

	err = l.writeScript(filepath.Join(rootfs, defaultInitScriptPath), config)
	if err != nil{
		return err
	}

	err = l.setRcLevel(filepath.Join(rootfs, "/etc/rc.d"))
	if err != nil {
		return err
	}

	spec.Process.Args = []string {initBinPath}

	return nil
}

type initScriptConfig struct {
	startCmd	string
}

const scriptDescription = `
#!/bin/sh
#****************************************************************#
# ScriptName: richContainer.sh
# Author: Pouch
# Create Date: 2018-1-29
# Modify Author:
# Modify Date: 2018-1-29
# Function: Rich Container Init Script
#***************************************************************#
`

const scriptStartCmd = `
start () {
	$start
}
`
const scriptStopCmd = `
stop () {
	echo "not set stop cmd"
}
`

const scriptC = `
case "$1" in
  start)
		start
        ;;
  stop)
		stop
        ;;
  status)
        echo "not set"
        ;;
  restart|reload)
        stop
        sleep 1
        start
        ;;
  *)
        echo $"Usage: $0 {start|stop|status|restart|reload}"
        exit 3
esac

exit $?
`

func (l *initdLauncher) writeScript(filePath string, config *initScriptConfig) error {
	dir := filepath.Dir(filePath)

	err := os.MkdirAll(dir, 0x755)
	if err != nil{
		return err
	}

	f,err := os.OpenFile(filePath, os.O_CREATE | os.O_TRUNC | os.O_WRONLY, 0x755)
	if err != nil{
		return err
	}

	defer f.Close()

	//write script
	f.WriteString(scriptDescription)

	startCmd := strings.Replace(scriptStartCmd, "$start", config.startCmd, -1)
	f.WriteString(startCmd)

	f.WriteString(scriptStopCmd)
	f.WriteString(scriptC)

	return nil
}

//set runlevel 2-5 auto start
func (l *initdLauncher) setRcLevel(rcDir string) error {
	var(
		startIndex int = 60
		endIndex int =99

		rcStart int = 2
		rcEnd int = 5
	)

	currentDir,err := os.Getwd()
	if err != nil {
		return err
	}

	defer os.Chdir(currentDir)

	for i:= rcStart; i <= rcEnd; i ++ {
		dir := filepath.Join(rcDir, fmt.Sprintf("rc%d.d", i))
		fInfos,err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		names := []string {}

		for _,f := range fInfos {
			if f.Mode() & os.ModeSymlink != 0 {
				names = append(names, f.Name())
			}
		}

		activeIndex,err := l.getActiveIndex(names, startIndex, endIndex)
		if err != nil {
			return err
		}

		//create link to /etc/rc.d/init.d/richContainer
		err = os.Chdir(dir)
		if err != nil {
			return err
		}

		//create link
		err = exec.Command("ln", "-s", fmt.Sprintf("../init.d/%s", defaultInitScriptName),
			fmt.Sprintf("S%d%s", activeIndex, defaultInitScriptName)).Run()

		if err != nil {
			return err
		}
	}

	return nil
}

//the function is to find active index in boot init
//it has some bugs if index is in (0,9) and names has (1,9)x prefix, but the start index is more than 50
func (l *initdLauncher) getActiveIndex(names []string, startIndex int, endIndex int) (int, error) {
	for i:= startIndex; i <= endIndex; i ++ {
		confict := false
		for _,name := range names {
			if strings.HasPrefix(name, fmt.Sprintf("K%d", i)) {
				confict = true
				break
			}

			if strings.HasPrefix(name, fmt.Sprintf("S%d", i)) {
				confict = true
				break
			}
		}

		if !confict{
			return i, nil
		}
	}

	return -1, errors.New("not found start index in rc")
}