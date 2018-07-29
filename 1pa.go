package main

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/vinc3m1/opvault"
)

func main() {
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

	profile, err := vault.Profile("default")

	if err != nil {
		fmt.Printf("Error opening profile [%s]: %s\n", profileName, err)
		os.Exit(1)
	}

	locked := true
	for locked {
		promptPassword := promptui.Prompt{
			Label: fmt.Sprintf("Password (hint: %s)", profile.PasswordHint()),
			Mask:  '*',
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

	fmt.Println("vault unlocked!")

	items, err := profile.Items()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Found %d items!\n", len(items))

	for i, item := range items {
		fmt.Printf("%d: item title: %s category: %s url: %s \n", i, item.Title(), item.Category(), item.Url())

		overview := item.Overview()
		overviewKeys := make([]string, 0, len(overview))
		for k := range overview {
			overviewKeys = append(overviewKeys, k)
		}
		fmt.Printf("    overview keys: %s\n", overviewKeys)
		fmt.Printf("    URLs: %s\n", item.Urls())

		data := item.Data()
		dataKeys := make([]string, 0, len(data))
		for k := range overview {
			dataKeys = append(dataKeys, k)
		}
		fmt.Printf("    data keys: %s\n", dataKeys)
		fmt.Printf("    URLs: %s\n", data["URLs"])

		detail, _ := item.Detail()
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

func printUsage() {
	fmt.Println(`Usage:
	1pa [vault]`)
}
