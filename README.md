# html5up-to-hugo

WORK IN PROGRESS

Here may come a common convertor and the result of converting all (or the best) of [HTML5Up](https://html5up.net/)´s themes.

The original themes are licensed with The Creative Commons Attribution 3.0 License, see https://html5up.net/license


# Build the Themes

```bash
# Only needed once (or if a new theme is added etc.)
go run main.go get

 go run main.go build
 cd exampleSite
 HUGO_THEME=lens hugo server
 ```


 # Strategy

* Put the common layouts into `template/layouts`
* Put the specific (use same name to overwrite) templates inside `template/[theme]/layouts/`
* We will probably need more templating support. Let us see.

**And try to keep it as DRY as possible.**

