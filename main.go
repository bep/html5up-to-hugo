package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	//"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hacdias/fileutils"

	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/mholt/archiver"
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
	Name        string
	Description string
}

func (t theme) Title(in string) string {
	return strings.Title(in)
}

func main() {
	//getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	args := os.Args

	cmd := "build"
	if len(args) > 1 {
		cmd = args[1]
		args = args[1:]
	}

	switch cmd {
	case "get":
		failOnError(get)
	case "build":
		failOnError(build)
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

func build() error {
	b, err := newThemeBuilder()
	if err != nil {
		return err
	}

	return b.build()
}

func (b *themeBuilder) build() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	buildPath := filepath.Join(pwd, "build")
	templateDir := filepath.Join(pwd, "template")

	if err := os.RemoveAll(buildPath); err != nil {
		return err
	}

	for _, theme := range b.cfg.Themes {
		themeBuildPath := filepath.Join(buildPath, theme.Name)
		os.MkdirAll(themeBuildPath, 0755)

		themeTOMLTpl, err := ioutil.ReadFile(filepath.Join(templateDir, "theme.toml"))
		if err != nil {
			return err
		}

		tpl, err := template.New("").Parse(string(themeTOMLTpl))
		if err != nil {
			return err
		}

		var out bytes.Buffer
		if err := tpl.Execute(&out, theme); err != nil {
			return err
		}

		ioutil.WriteFile(filepath.Join(themeBuildPath, "theme.toml"), out.Bytes(), 0755)

		assetsPath := filepath.Join(themeBuildPath, "static", "assets")
		srcAssetsPath := filepath.Join(pwd, "temp", "download", theme.Name, "assets")

		layoutsPath := filepath.Join(themeBuildPath, "layouts")

		os.MkdirAll(assetsPath, 0755)

		if err := fileutils.CopyDir(filepath.Join(templateDir, "layouts"), layoutsPath); err != nil {
			return err
		}

		for _, dirname := range []string{"css", "fonts", "js"} {
			src := filepath.Join(srcAssetsPath, dirname)
			destination := filepath.Join(assetsPath, dirname)
			if err := fileutils.CopyDir(src, destination); err != nil {
				return err
			}
		}
	}

	return nil
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
