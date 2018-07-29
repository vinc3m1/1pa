package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vinc3m1/opvault"
)

var (
	cfgFile, cfgFileUsed string
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.1pa.yaml)")
	// viper.SetDefault("vault", "~/Dropbox/1Password/1Password.opvault")
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".1pa" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".1pa")
	}

	if err := viper.ReadInConfig(); err == nil {
		cfgFileUsed = viper.ConfigFileUsed()
	}
}

var rootCmd = &cobra.Command{
	Use:     "1pa [vault]",
	Short:   "1pa is a command line interface to 1password",
	Long:    "1pa is a command line interface to 1password",
	Version: "0.1",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var vaultPath string
		if len(args) > 0 {
			vaultPath = args[0]
			if vaultPath != "" {
				fmt.Printf("Opening vault: %s\n", vaultPath)
			}
		}
		if vaultPath == "" && cfgFileUsed != "" {
			vaultPath = viper.GetString("vault")
			if vaultPath != "" {
				fmt.Printf("Using config file: %s\n", cfgFileUsed)
				fmt.Printf("Opening vault: %s\n", vaultPath)
			}
		}
		if vaultPath == "" {
			fmt.Printf("Specify a vault or configure the vault path globally with 1pa config vault [vault]\n")
			os.Exit(1)
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

		profile, err := vault.Profile(profileName)

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
			fmt.Printf("    URLs: %s\n", overview["URLs"])

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

	},
}

// Execute the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
