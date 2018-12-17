package cmd

import (
	"path/filepath"

	"fmt"

	"github.com/privacybydesign/irmago"
	"github.com/privacybydesign/irmago/internal/fs"
	"github.com/spf13/cobra"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify irma_configuration_path",
	Short: "Verify irma_configuration folder correctness and authenticity",
	Long:  `The verify command parses the specified irma_configuration folder and checks the signatures of the contained scheme managers.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := RunVerify(args[0])
		if err == nil {
			fmt.Println()
			fmt.Println("Verification was successful.")
		}
		return err
	},
}

func RunVerify(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	isScheme, err := fs.PathExists(filepath.Join(path, "index"))
	if err != nil {
		return err
	}
	if !isScheme {
		fmt.Println("No index file found; verifying subdirectories")
		return VerifyIrmaConfiguration(path)
	} else {
		fmt.Println("Verifying scheme " + filepath.Base(path))
		return VerifyScheme(path)
	}
}

func VerifyScheme(path string) error {
	conf, err := irma.NewConfiguration(filepath.Dir(path), "")
	if err != nil {
		return err
	}

	return conf.ParseSchemeManagerFolder(path, irma.NewSchemeManager(filepath.Base(path)))
}

func VerifyIrmaConfiguration(path string) error {
	if filepath.Base(path) != "irma_configuration" {
		fmt.Printf("Notice: specified folder name is '%s'; when using in IRMA applications it should be called 'irma_configuration'\n", filepath.Base(path))
	}

	conf, err := irma.NewConfiguration(path, "")
	if err != nil {
		return err
	}
	if err := conf.ParseFolder(); err != nil {
		return err
	}
	if err := conf.CheckKeys(); err != nil {
		return err
	}

	for _, manager := range conf.SchemeManagers {
		if err := conf.VerifySchemeManager(manager); err != nil {
			return err
		}
	}

	for _, warning := range conf.Warnings {
		fmt.Println("Warning: " + warning)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(verifyCmd)
}
