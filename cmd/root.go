package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/miquella/opvault"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var vaultPath string
		if len(args) > 0 {
			vaultPath = args[0]
			if vaultPath != "" {
				fmt.Printf("Opening vault: %q\n", vaultPath)
			}
		}
		if vaultPath == "" && cfgFileUsed != "" {
			vaultPath = viper.GetString("vault")
			if vaultPath != "" {
				fmt.Printf("Using config file: %q\n", cfgFileUsed)
				fmt.Printf("Opening vault: %q\n", vaultPath)
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
			fmt.Printf("Opening profile: %q\n", profileName)
		}

		profile, err := vault.Profile(profileName)

		if err != nil {
			fmt.Printf("Error opening profile [%q]: %q\n", profileName, err)
			os.Exit(1)
		}

		locked := true
		for locked {
			promptPassword := promptui.Prompt{
				Label: fmt.Sprintf("Password (hint: %q)", profile.PasswordHint()),
				Mask:  '*',
			}

			password, err := promptPassword.Run()

			if err != nil {
				os.Exit(1)
			}

			err = profile.Unlock(password)

			if err != nil {
				fmt.Println("Incorrect password")
				continue
			}
			locked = false
		}

		fmt.Println("vault unlocked!")
	},
}

// Execute the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
