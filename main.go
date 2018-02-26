package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"

	"github.com/BurntSushi/toml"
)

var (
	logger                *log.Logger = log.New(os.Stdout, "html5up-to-hugo: ", log.Ldate|log.Ltime|log.Lshortfile)
	themeDownloadTemplate             = "https://html5up.net/%s/download"
)

type themeBuilder struct {
	cfg config
}

type config struct {
	Themes []theme
}

type theme struct {
	Name string
}

func main() {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	args := os.Args

	cmd := "get"
	if len(args) > 1 {
		cmd = args[1]
		args = args[1:]
	}

	switch cmd {
	case "get":
		getCmd.Parse(args)
		failOnError(get)
	default:
		fmt.Printf("%q is not valid command.\n", cmd)
		os.Exit(2)
	}
}

func failOnError(f func() error) {
	if err := f(); err != nil {
		log.Fatal(err)
	}
}

func get() error {
	b, err := newThemeBuilder()
	if err != nil {
		return err
	}

	return b.get()
}

func (b *themeBuilder) get() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	baseDownloadPath := filepath.Join(pwd, "temp", "download")
	log.Println("Download themes to", baseDownloadPath)

	for _, theme := range b.cfg.Themes {
		downloadPath := filepath.Join(baseDownloadPath, theme.Name)
		downloadURL := fmt.Sprintf(themeDownloadTemplate, theme.Name)

		os.MkdirAll(downloadPath, 0755)

		resp, err := http.Get(downloadURL)
		if err != nil {
			return err
		}

		err = archiver.Zip.Read(resp.Body, downloadPath)

		resp.Body.Close()

		if err != nil {
			return err
		}

	}

	return nil
}

func newThemeBuilder() (*themeBuilder, error) {
	b := &themeBuilder{}

	conf, err := readConfig()
	if err != nil {
		return nil, err
	}

	b.cfg = conf

	return b, nil
}

func readConfig() (config, error) {
	var conf config
	f, err := os.Open("config.toml")
	if err != nil {
		return conf, err
	}
	defer f.Close()

	_, err = toml.DecodeReader(f, &conf)
	if err != nil {
		return conf, err
	}

	return conf, err

}
