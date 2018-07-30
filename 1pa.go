package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/vinc3m1/opvault"
)

func main() {
	// validate command line arguments
	args := os.Args[1:]
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

	// printDebug(&items)

	prompt := promptui.Select{
		Label: "Choose an item",
		Items: items,
		Size:  20,
		Searcher: func(input string, index int) bool {
			item := items[index]

			var buffer bytes.Buffer
			buffer.WriteString(item.Title())
			for _, url := range item.Urls() {
				buffer.WriteString(url.Url())
			}
			input = strings.ToLower(input)

			return strings.Contains(strings.ToLower(buffer.String()), strings.ToLower(input))
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   `▸ {{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ printf "%.80s" .Url | cyan }}`,
			Inactive: `  {{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ printf "%.80s" .Url | cyan }}`,
			Selected: `{{ if .Trashed }}{{ "[Deleted] " | red }}{{ end }}{{ printf "[%s]" .Category.String | blue }} {{ .Title }} {{ printf "%.80s" .Url | cyan }}`,
			Details: `
------------ Item ------------
{{ "Name:" | faint }}    {{ .Title }}
{{- range $i, $url := .Urls }}
	{{- if eq $url.Label "" }}
		{{- "\nwebsite: " | faint}}
	{{- else }}
		{{- printf "\n%s: " $url.Label | faint }}
	{{- end }}
	{{- $url.Url }}
{{- end }}
{{- with .Detail }}
	{{- range $i, $field := .Fields }}
		{{- if ne $field.Designation "" }}
			{{- printf "\n%s:" $field.Designation | faint }} {{ if eq $field.Type "P" }}********{{ else }}{{ $field.Value }}{{ end }}
		{{- end }}
	{{- end }}
	{{- range $i, $section := .Sections }}
			{{- if and (ne $section.Title "") (gt (len $section.Fields) 0) }}
				{{- printf "\n[%s]" $section.Title | faint }}
			{{- end }}
			{{- range $j, $sectionField := $section.Fields }}
				{{- if ne $sectionField.Value "" }}
					{{- printf "\n%s: " $sectionField.Title | faint }}
					{{- if eq $sectionField.Kind "concealed" }}********{{- else }}{{ $sectionField.Value }}{{ end }}
				{{- end }}
			{{- end }}
	{{- end }}
	{{- if ne .Notes "" }}trashe
		{{- "\nNotes:" | faint }} {{ .Notes }}
	{{- end }}
{{- end }}`,
		},
	}

	i, item, err := prompt.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("You chose [%d] %s\n", i, item)
}

func printUsage() {
	fmt.Println()
	fmt.Println(`Usage:
    1pa [vault]`)
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
