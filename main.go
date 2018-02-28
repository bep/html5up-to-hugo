package main

import (
	"bytes"
	"io/ioutil"
	"path"
	"strings"
	//"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gohugoio/hugo/commands"

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
	clean bool
	cfg   config
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

	b, err := newThemeBuilder()
	if err != nil {
		log.Fatal(err)
	}

	cmd := "build"
	if len(args) > 1 {
		cmd = args[1]
		args = args[1:]
	}

	switch cmd {
	case "get":
		failOnError(b.get)
	case "build":
		failOnError(b.build)
		failOnError(b.preview)
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

func (b *themeBuilder) build() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	buildPath := filepath.Join(pwd, "build")
	templateDir := filepath.Join(pwd, "template")

	if b.clean {
		if err := os.RemoveAll(buildPath); err != nil {
			return err
		}
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

		// First copy the common base templates.
		if err := fileutils.CopyDir(filepath.Join(templateDir, "layouts"), layoutsPath); err != nil {
			return err
		}

		// Then overwrite with the specific theme templates.
		if err := fileutils.CopyDir(filepath.Join(templateDir, "themes", theme.Name, "layouts"), layoutsPath); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
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

func (b *themeBuilder) preview() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	exampleSite := filepath.Join(pwd, "exampleSite")
	buildPath := filepath.Join(pwd, "preview")
	templateDir := filepath.Join(pwd, "template", "preview")

	if b.clean {
		if err := os.RemoveAll(buildPath); err != nil {
			return err
		}
	}

	staticDir := filepath.Join(buildPath, "static")
	contentDir := filepath.Join(buildPath, "content")

	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(contentDir, 0755)

	for _, theme := range b.cfg.Themes {
		if err := buildHugoSite(exampleSite, staticDir, theme.Name); err != nil {
			return err
		}
	}

	if err := fileutils.CopyFile(filepath.Join(templateDir, "config.toml"), filepath.Join(buildPath, "config.toml")); err != nil {
		return err
	}

	contentTpl := `---
title: %q
description: %q
weight: %d
---

`

	if err := ioutil.WriteFile(filepath.Join(contentDir, "_index.md"), []byte(`---
title: "Theme Previews"
---

This is a work in progress. The goal is to create unified set of Hugo themes from https://html5up.net/

`), 0755); err != nil {
		return err
	}

	if err := fileutils.CopyFile(filepath.Join(templateDir, "images", "post-1.jpg"), filepath.Join(contentDir, "featured.jpg")); err != nil {
		return err
	}

	for i, theme := range b.cfg.Themes {
		bundleDir := filepath.Join(contentDir, "sect", theme.Name)
		if err := os.MkdirAll(bundleDir, 0777); err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.Join(bundleDir, "index.md"),
			[]byte(fmt.Sprintf(contentTpl, strings.Title(theme.Name), theme.Description, i+1)), 0755); err != nil {
			return err
		}

		image := filepath.Join(templateDir, "images", fmt.Sprintf("post-%d.jpg", i+1))
		if err := fileutils.CopyFile(image, filepath.Join(bundleDir, "featured.jpg")); err != nil {
			return err
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

func buildHugoSite(source, destination, theme string) error {
	defer commands.Reset()

	destination = path.Join(destination, "theme", theme)

	baseURL := fmt.Sprintf("http://localhost:1313/theme/%s/", theme)
	flags := []string{"--quiet", "--gc",
		fmt.Sprintf("--baseURL=%s", baseURL),
		fmt.Sprintf("--source=%s", source),
		fmt.Sprintf("--destination=%s", destination),
		fmt.Sprintf("--theme=%s", theme)}
	os.Args = []string{os.Args[0]}

	if err := commands.HugoCmd.ParseFlags(flags); err != nil {
		log.Fatal(err)
	}

	if _, err := commands.HugoCmd.ExecuteC(); err != nil {
		return err

	}

	return nil
}
