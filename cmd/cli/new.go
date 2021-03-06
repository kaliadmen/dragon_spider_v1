package main

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var appURL string

func createApp(appName string) error {
	appName = strings.ToLower(appName)
	appName = strings.TrimSpace(appName)

	//sanitize application name
	if strings.ContainsAny("/", appName) {
		appURL = appName
		exploded := strings.SplitAfter(appName, "/")
		appName = exploded[len(exploded)-1]
	}
	if strings.ContainsAny(appName, "~`!@#$%^&*()+={}[]|\\:;'\",<>?") {
		return errors.New("invalid application name entered")
	}
	if strings.Contains(appName, " ") {
		appName = strings.ReplaceAll(appName, " ", "_")
	}
	if appURL == "" {
		appURL = appName
	}

	//git clone bare application
	color.Green("\tCloning from git repository")
	_, err := git.PlainClone("./"+appName, false, &git.CloneOptions{
		URL:      "https://github.com/kaliadmen/dragon_spider_skeleton.git",
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		return err
	}

	//remove .git directory
	err = os.RemoveAll(fmt.Sprintf("./%s/.git", appName))
	if err != nil {
		gracefulExit(err)
	}

	//create a evn file
	color.Yellow("\tCreating .env file...")
	data, err := templateFs.ReadFile("templates/env.txt")
	if err != nil {
		gracefulExit(err)
	}

	env := string(data)
	env = strings.ReplaceAll(env, "${APP_NAME}", appName)
	env = strings.ReplaceAll(env, "${APP_GITHUB_URL}", appURL)
	env = strings.ReplaceAll(env, "${KEY}", ds.RandomString(32))

	err = copyDataToFile([]byte(env), fmt.Sprintf("./%s/.env", appName))
	if err != nil {
		gracefulExit(err)
	}

	//create makefile
	var source *os.File

	if runtime.GOOS == "windows" {
		source, err = os.Open(fmt.Sprintf("./%s/Makefile.windows", appName))
		if err != nil {
			gracefulExit(err)
		}
	} else {
		source, err = os.Open(fmt.Sprintf("./%s/Makefile.linux", appName))
		if err != nil {
			gracefulExit(err)
		}
	}

	defer func(source *os.File) {
		err := source.Close()
		if err != nil {
			gracefulExit(err)
		}
	}(source)

	dest, err := os.Create(fmt.Sprintf("./%s/Makefile", appName))
	if err != nil {
		gracefulExit(err)
	}

	defer func(dest *os.File) {
		err := dest.Close()
		if err != nil {
			gracefulExit(err)
		}
	}(dest)

	_, err = io.Copy(dest, source)
	if err != nil {
		gracefulExit(err)
	}

	_ = os.Remove("./" + appName + "/Makefile.linux")
	_ = os.Remove("./" + appName + "/Makefile.windows")

	//update go.mod
	color.Yellow("\tCreating a go.mod file...")
	_ = os.Remove("./" + appName + "/go.mod")

	data, err = templateFs.ReadFile("templates/go.mod.txt")
	if err != nil {
		gracefulExit(err)
	}
	modFile := string(data)
	modFile = strings.ReplaceAll(modFile, "${APP_NAME}", appURL)

	err = copyDataToFile([]byte(modFile), "./"+appName+"/go.mod")
	if err != nil {
		gracefulExit(err)
	}

	//update existing go files with correct names and imports
	color.Yellow("\tUpdating source files...")
	err = os.Chdir("./" + appName)
	if err != nil {
		gracefulExit(err)
	}
	updateSource()

	//run go mod tidy
	color.Yellow("\tRunning go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	err = cmd.Start()
	if err != nil {
		gracefulExit(err)
	}
	color.Green("Done building " + appURL)
	color.Green("Go build something!")

	return nil
}
