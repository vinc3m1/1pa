package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/vinc3m1/opvault"
)

type itemWrapper struct {
	*opvault.Item
	ShowPass bool
}

func main() {
	// flags
	showPassPtr := flag.Bool("s", false, "Show password fields.")
	flag.Parse()

	// validate command line arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Please specify a vault.")
		printUsage()
		os.Exit(1)
	} else if len(args) > 1 {
		fmt.Println("Too many arguments, please specify just one vault.")
		printUsage()
		os.Exit(1)
	}

	// expand any '~' in the directory and open the vault
	vaultPath := args[0]
	if vaultPath != "" {
		fmt.Printf("Opening vault: %s\n", vaultPath)
	}

	expandedVaultPath, err := homedir.Expand(vaultPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	vault, err := opvault.Open(expandedVaultPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	profiles, err := vault.ProfileNames()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(profiles) == 0 {
		fmt.Println("not a valid vault")
		os.Exit(1)
	}

	// the only profile should be 'default', but show a chooser if there is more than 1 for some reason
	var profileName string
	if len(profiles) == 1 {
		profileName = profiles[0]
	} else {
		promptProfile := promptui.Select{
			Label: "Select Profile (usually default)",
			Items: profiles,
		}
		_, profileName, err := promptProfile.Run()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Opening profile: %s\n", profileName)
	}

	// open selected profile
	profile, err := vault.Profile(profileName)
	if err != nil {
		fmt.Printf("Error opening profile [%s]: %s\n", profileName, err)
		os.Exit(1)
	}

	// prompt for password and unlock vault
	locked := true
	for locked {
		promptPassword := promptui.Prompt{
			Label: fmt.Sprintf("Password (hint: %s)", profile.PasswordHint()),
			Mask:  '*',
			Validate: func(input string) error {
				if len(input) < 1 {
					return errors.New("Password cannot be empty")
				}
				return nil
			},
		}
		password, err := promptPassword.Run()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = profile.Unlock(password)
		if err != nil {
			fmt.Println("wrong password, try again")
			continue
		}
		locked = false
	}

	// get all items in the vault
	items, err := profile.Items()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// sort items by category, then name
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Trashed() != right.Trashed() {
			return right.Trashed()
		}
		if left.Category() != right.Category() {
			return left.Category() < right.Category()
		}
		return left.Title() < right.Title()
	})

	itemWrappers := make([]*itemWrapper, len(items))
	for idx, item := range items {
		itemWrappers[idx] = &itemWrapper{item, *showPassPtr}
	}

	// printDebug(&items)

	var funcMap = promptui.FuncMap

	funcMap["trimNewlines"] = newlinesToSpaces
	funcMap["normalize"] = normalizeNewlines

	prompt := promptui.Select{
		Label: "Choose an item to copy password to clipboard",
		Items: itemWrappers,
		Size:  10,
		Searcher: func(input string, index int) bool {
			item := itemWrappers[index]

			var buffer bytes.Buffer
			buffer.WriteString(item.Title())
			for _, url := range item.Urls() {
				buffer.WriteString(url.Url())
			}
			detail, _ := item.Detail()
			for _, field := range detail.Fields() {
				if *showPassPtr || field.Type() != opvault.PasswordFieldType {
					buffer.WriteString(field.Value())
				}
			}
			buffer.WriteString(detail.Notes())
			input = strings.ToLower(input)

			return strings.Contains(strings.ToLower(buffer.String()), strings.ToLower(input))
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}:",
			Active:   `▸ {{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ .Info | trimNewlines | faint }}`,
			Inactive: `  {{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ .Info | trimNewlines | faint }}`,
			Selected: `{{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ .Info | trimNewlines | faint }}`,
			Details: `------------ Item ------------
				{{- "\nName:" | faint }} {{ .Title }}
				{{- range .Urls }}
					{{- if eq .Label "" }}
						{{- "\nwebsite: " | faint}}
					{{- else }}
						{{- printf "\n%s: " .Label | faint }}
					{{- end }}
					{{- printf "%.150s" .Url }}
				{{- end }}
				{{- with .Detail }}
					{{- range .Fields }}
						{{- if ne .Designation "" }}
							{{- printf "\n%s:" .Designation | faint }} {{ if and (not $.ShowPass) (or (eq .Type "P") (eq .Designation "password")) }}********{{ else }}{{ .Value | normalize }}{{ end }}
						{{- end }}
					{{- end }}
					{{- range .Sections }}
							{{- if and (ne .Title "") (gt (len .Fields) 0) }}
								{{- printf "\n[%s]" .Title | faint }}
							{{- end }}
							{{- range .Fields }}
								{{- if ne .Value "" }}
									{{- printf "\n%s: " .Title | faint }}
									{{- if and (not $.ShowPass) (eq .Kind "concealed") }}********{{- else }}{{ .Value | normalize }}{{ end }}
								{{- end }}
							{{- end }}
					{{- end }}
					{{- if ne .Notes "" }}
						{{- "\nNotes:" | faint }} {{ .Notes | normalize}}
					{{- end }}
				{{- end }}`,
			FuncMap: funcMap,
		},
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	item := items[i]

	detail, _ := item.Detail()
	for _, field := range detail.Fields() {
		if field.Designation() == opvault.PasswordDesignation {
			err = clipboard.WriteAll(field.Value())
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("password copied to clipboard")
			return
		}
	}
}

func printUsage() {
	fmt.Println()
	fmt.Println(`Usage:
    1pa [-s] <vault>`)
	fmt.Println()
}

func printDebug(items *[]*opvault.Item) {
	for i, item := range *items {
		fmt.Printf("%d: item title: %s category: %s url: %s \n", i, item.Title(), item.Category(), item.Url())

		overview := item.Overview()
		overviewKeys := make([]string, 0, len(overview))
		for k := range overview {
			overviewKeys = append(overviewKeys, k)
		}
		fmt.Printf("    overview keys: %s\n", overviewKeys)
		fmt.Printf("    URLs: %s\n", item.Urls())
		fmt.Printf("    ps: %s\n", overview["ps"])
		fmt.Printf("    ainfo: %s\n", overview["ainfo"])

		data := item.Data()
		dataKeys := make([]string, 0, len(data))
		for k := range data {
			dataKeys = append(dataKeys, k)
		}
		fmt.Printf("    data keys: %s\n", dataKeys)
		fmt.Printf("    o: %s\n", data["o"])
		fmt.Printf("    d: %s\n", data["d"])
		fmt.Printf("    k: %s\n", data["k"])
		fmt.Printf("    tx: %s\n", data["tx"])
		fmt.Printf("    trashed bool: %t\n", item.Trashed())

		detail, _ := item.Detail()

		detailData := detail.Data()
		detailDataKeys := make([]string, 0, len(detailData))
		for k := range detailData {
			detailDataKeys = append(detailDataKeys, k)
		}
		fmt.Printf("    detailData keys: %s\n", detailDataKeys)

		for j, field := range detail.Fields() {
			fmt.Printf("    %d: field type: %s name: %s designation: %s\n", j, field.Type(), field.Name(), field.Designation())
		}
		for k, section := range detail.Sections() {
			fmt.Printf("    %d: section name: %q title: %q\n", k, section.Name(), section.Title())
			for l, sectionField := range section.Fields() {
				fmt.Printf("        %d: sectionField kind: %s name: %s title: %s\n", l, sectionField.Kind(), sectionField.Name(), sectionField.Title())
			}
		}
	}
}

func newlinesToSpaces(s string) string {
	d := []byte(s)
	// replace CR LF \r\n (windows) space
	d = bytes.Replace(d, []byte{13, 10}, []byte{32}, -1)
	// replace CF \r (mac) with space
	d = bytes.Replace(d, []byte{13}, []byte{32}, -1)
	// replace LF \n (unix) with space
	d = bytes.Replace(d, []byte{10}, []byte{32}, -1)
	return string(d[:])
}

func normalizeNewlines(s string) string {
	d := []byte(s)
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return string(d[:])
}
