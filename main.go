package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Add ability to clean
// Add help
// Add dependency on other section

var manifest = flag.String("manifest", "", "the manifest to use")

type Manifest struct {
	Actions         Actions  `json:"actions"`
	Sections        Sections `json:"sections"`
	ProcessSections []string `json:"process-sections"`
}
type Action struct {
	Args   []string `json:"args"`
	Verify string   `json:"verify"`
	Do     string   `json:"do"`
}
type Unit struct {
	Name         string            `json:"name"`
	Ensure       []string          `json:"ensure"`
	Definition   map[string]string `json:"definition"`
	Prerequisite []string          `json:"prerequisite"`
	OnChange     []string          `json:"on-change"`
}
type Actions map[string]map[string]Action
type Sections map[string][]Unit

var actions map[string]map[string]string

func main() {
	flag.Parse()

	if "" == *manifest {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	fmt.Printf("Parsing manifest %s\n", *manifest)
	loadedManifest, err := loadManifest(manifest)
	if nil != err {
		fmt.Printf("Parsing manifest failes with: %s\n", err)
		return
	}

	fmt.Println("")

	fmt.Printf("Registering actions\n")
	err = registerActions(loadedManifest)
	if nil != err {
		fmt.Printf("Registering actions failed with: %s\n", err)
		return
	}

	fmt.Println("")

	for _, section := range (*loadedManifest).ProcessSections {
		fmt.Printf("Processing section %s\n", section)
		err = processSection(section, loadedManifest.Sections[section], actions)
		if nil != err {
			fmt.Printf("Processing section failed with: %s\n", err)
			return
		}
		fmt.Println("")
	}
}

func loadManifest(file *string) (manifest *Manifest, err error) {
	data, err := ioutil.ReadFile(*file)
	if nil != err {
		return
	}
	manifest = new(Manifest)
	err = json.Unmarshal(data, manifest)
	return
}

func registerActions(manifest *Manifest) (err error) {
	actions = make(map[string]map[string]string)
	for section, units := range manifest.Sections {
		for _, unit := range units {
			unitName := unit.Name
			unitDef := unit.Definition

			for action, subaction := range manifest.Actions[section] {
				key := fmt.Sprintf("%s.%s.%s", section, unitName, action)

				if _, ok := actions[key]; ok {
					continue
				}

				args := make([]string, 0, 0)
				args = append(args, unitName)

				for _, a := range subaction.Args {
					args = append(args, unitDef[a])
				}

				do := subaction.Do
				for i, a := range args {
					do = strings.Replace(do, fmt.Sprintf("ARG%d", i), a, -1)
				}

				verify := subaction.Verify
				for i, a := range args {
					verify = strings.Replace(verify, fmt.Sprintf("ARG%d", i), a, -1)
				}

				actions[key] = map[string]string{
					"do":     do,
					"verify": verify,
				}
			}
		}
	}
	return
}

func processSection(sectionName string, section []Unit, actions map[string]map[string]string) (err error) {
	for _, unit := range section {
		unitName := unit.Name
		change := false

		for _, a := range unit.Prerequisite {
			fmt.Printf("\tProcessing prerequisites for %s\n", unitName)
			if !strings.Contains(a, ".") {
				a = fmt.Sprintf("%s.%s.%s", sectionName, unitName, a)
			}
			_, err = doIt(actions[a], fmt.Sprintf("\t\tPrerequisite %s", a))
			if nil != err {
				fmt.Printf("\tProcessing prerequisites failed with: %s\n", err)
				return
			}
		}

		for _, a := range unit.Ensure {
			fmt.Printf("\tProcessing ensures for %s\n", unitName)
			if !strings.Contains(a, ".") {
				a = fmt.Sprintf("%s.%s.%s", sectionName, unitName, a)
			}
			_change := false
			_change, err = doIt(actions[a], fmt.Sprintf("\t\tEnsure %s", a))
			if _change {
				change = _change
			}
			if nil != err {
				fmt.Printf("\tProcessing ensures failed with %s\n", err)
				return
			}
		}

		if change {
			for _, a := range unit.OnChange {
				fmt.Printf("\tProcessing on-changes for %s\n", unitName)
				if !strings.Contains(a, ".") {
					a = fmt.Sprintf("%s.%s.%s", sectionName, unitName, a)
				}
				_, err = doIt(actions[a], fmt.Sprintf("\t\tOnChange %s", a))
				if nil != err {
					fmt.Printf("\tProcessing on-changes failed with: %s\n", err)
					return
				}
			}
		}

		fmt.Println("")
	}

	return
}

func execCmd(_cmd string) (status int, out string, err error) {
	fmt.Printf("\t\t\t%s\n", _cmd)
	cmd := exec.Command("/bin/bash", "-c", _cmd)
	var waitStatus syscall.WaitStatus
	_out, err := cmd.CombinedOutput()
	out = string(_out)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			status, _ = strconv.Atoi(string([]byte(fmt.Sprintf("%d", waitStatus.ExitStatus()))))
		}
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		status, _ = strconv.Atoi(string([]byte(fmt.Sprintf("%d", waitStatus.ExitStatus()))))
	}
	return
}

func doIt(action map[string]string, msg string) (change bool, err error) {
	do := action["do"]
	status := 1
	verify := ""
	if v, ok := action["verify"]; ok {
		verify = v
	}

	// Should I?
	if verify != "" {
		status, _, err = execCmd(verify)
	}
	if 0 == status {
		fmt.Printf("%s: VERIFIED no further action required %d\n", msg, status)
	} else {
		// Do it
		fmt.Printf("%s: ACTION REQUIRED %d\n", msg, status)
		status, _, err = execCmd(do)
		if 0 != status {
			fmt.Printf("%s: ACTION FAILED %d %s\n", msg, status, err)
			return
		}
		change = true

		// Verify
		if verify != "" {
			status, _, err = execCmd(verify)
			if 0 != status {
				fmt.Printf("%s: VERIFYING ACTION FAILED %d %s\n", msg, status, err)
				return
			}
			fmt.Printf("%s: VERIFYING ACTION SUCCEDED %d\n", msg, status)
		} else {
			status = 0
			fmt.Printf("%s: VERIFYING ACTION SKIPPED %d\n", msg, status)
		}
	}
	return
}
