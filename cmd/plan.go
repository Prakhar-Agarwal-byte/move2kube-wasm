package cmd

import (
	"context"
	"io/fs"
	"os"

	// "os/signal"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"

	// "github.com/konveyor/move2kube/common/download"
	// "github.com/konveyor/move2kube/common/vcs"
	"github.com/konveyor/move2kube-wasm/lib"
	// "github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube-wasm/types/plan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type planFlags struct {
	progressServerPort    int
	planfile              string
	srcpath               string
	name                  string
	customizationsPath    string
	transformerSelector   string
	disableLocalExecution bool
	failOnEmptyPlan       bool
	//Configs contains a list of config files
	configs []string
	//Configs contains a list of key-value configs
	setconfigs []string
	//PreSets contains a list of preset configurations
	preSets []string
}

func planHandler(cmd *cobra.Command, flags planFlags) {
	ctx, _ := context.WithCancel(cmd.Context())
	// logrus.AddHook(common.NewCleanupHook(cancel))
	// logrus.AddHook(common.NewCleanupHook(lib.Destroy))
	// ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	// go func() {
	// 	<-ctx.Done()
	// 	lib.Destroy()
	// 	stop()
	// 	common.Interrupt()
	// }()
	// defer lib.Destroy()

	var err error
	planfile := flags.planfile
	srcpath := flags.srcpath
	name := flags.name
	// isRemotePath := vcs.IsRemotePath(srcpath)
	// Check if the default customization folder exists in the working directory.
	// If not, skip the customization option
	if !cmd.Flags().Changed(customizationsFlag) {
		if _, err := os.Stat(common.DefaultCustomizationDir); err == nil {
			flags.customizationsPath = common.DefaultCustomizationDir
			// make all path(s) absolute
			flags.customizationsPath, err = filepath.Abs(flags.customizationsPath)
			if err != nil {
				logrus.Fatalf("Failed to make the customizations directory path %q absolute. Error: %q", flags.customizationsPath, err)
			}
		}
	}
	// Check if the default configuration file exists in the working directory.
	// If not, skip the configuration option
	if !cmd.Flags().Changed(configFlag) {
		if _, err := os.Stat(common.DefaultConfigFilePath); err == nil {
			flags.configs = []string{common.DefaultConfigFilePath}
		}
	}
	// make all path(s) absolute
	for i, c := range flags.configs {
		// if !download.IsRemotePath(c) {
		if c, err := filepath.Abs(c); err != nil {
			logrus.Fatalf("failed to make the config file path %s absolute. Error: %q", c, err)
		}
		flags.configs[i] = c
		// }
	}

	customizationsPath := flags.customizationsPath
	// Global settings
	common.DisableLocalExecution = flags.disableLocalExecution
	// Global settings

	planfile, err = filepath.Abs(planfile)
	if err != nil {
		logrus.Fatalf("Failed to make the plan file path %q absolute. Error: %q", planfile, err)
	}
	var fi fs.FileInfo
	// if srcpath != "" && !isRemotePath {
	if srcpath != "" {
		srcpath, err = filepath.Abs(srcpath)
		if err != nil {
			logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
		}
		fi, err = os.Stat(srcpath)
		if err != nil {
			logrus.Fatalf("Unable to access source directory : %s", err)
		}
		if !fi.IsDir() {
			logrus.Fatalf("Input is a file, expected directory: %s", srcpath)
		}
	}
	{
		logrus.Infof("before os.Stat on plan file")
		fs, err := os.ReadDir(".")
		if err != nil {
			panic(err)
		}
		for i, f := range fs {
			logrus.Infof("file[%d] %+v", i, f)
		}
	}
	logrus.Infof("planfile: '%s'", planfile)
	fi, err = os.Stat(planfile)
	if err == nil && fi.IsDir() {
		planfile = filepath.Join(planfile, common.DefaultPlanFile)
		_, err = os.Stat(planfile)
	}
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Warnf("the plan file doesn't exist")
			if strings.HasSuffix(planfile, string(os.PathSeparator)) {
				planfile = filepath.Join(planfile, common.DefaultPlanFile)
			} else if !strings.Contains(filepath.Base(planfile), ".") {
				planfile = filepath.Join(planfile, common.DefaultPlanFile)
			}
		} else {
			// logrus.Warnf("failed to stat. file info: %+v", fi)
			logrus.Fatalf("failed to access the plan file at path '%s' . Error: %q", planfile, err)
		}
	}

	// qaengine.StartEngine(true, 0, true)
	// qaengine.SetupConfigFile("", flags.setconfigs, flags.configs, flags.preSets, false)
	// if flags.progressServerPort != 0 {
	// 	startPlanProgressServer(flags.progressServerPort)
	// }
	p, err := lib.CreatePlan(ctx, srcpath, "", customizationsPath, flags.transformerSelector, name)
	if err != nil {
		logrus.Fatalf("failed to create the plan. Error: %q", err)
	}
	if err = plantypes.WritePlan(planfile, p); err != nil {
		logrus.Fatalf("failed to write the plan to file at path %s . Error: %q", planfile, err)
	}
	logrus.Debugf("Plan : %+v", p)
	logrus.Infof("Plan can be found at [%s].", planfile)
	if len(p.Spec.Services) == 0 && len(p.Spec.InvokedByDefaultTransformers) == 0 {
		if flags.failOnEmptyPlan {
			logrus.Fatalf("Did not detect any services in the directory %s . Also we didn't find any default transformers to run.", srcpath)
		}
		logrus.Warnf("Did not detect any services in the directory %s . Also we didn't find any default transformers to run.", srcpath)
	}
	{
		logrus.Infof("after planning, list files")
		fs, err := os.ReadDir(".")
		if err != nil {
			panic(err)
		}
		for i, f := range fs {
			logrus.Infof("file[%d] %+v", i, f)
		}
	}
}

// GetPlanCommand returns a command to do the planning
func GetPlanCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := planFlags{}
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan out a move",
		Long:  "Discover and create a plan file based on an input directory",
		Run:   func(cmd *cobra.Command, _ []string) { planHandler(cmd, flags) },
	}

	planCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory or a git url (see https://move2kube.konveyor.io/concepts/git-support).")
	planCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a file path to save plan to.")
	planCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	planCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory or a git url (see https://move2kube.konveyor.io/concepts/git-support) where customizations are stored. By default we look for "+common.DefaultCustomizationDir)
	planCmd.Flags().StringSliceVarP(&flags.configs, configFlag, "f", []string{}, "Specify config file locations. By default we look for "+common.DefaultConfigFilePath)
	planCmd.Flags().StringVarP(&flags.transformerSelector, transformerSelectorFlag, "t", "", "Specify the transformer selector.")
	planCmd.Flags().StringSliceVar(&flags.preSets, preSetFlag, []string{}, "Specify preset config to use.")
	planCmd.Flags().StringArrayVar(&flags.setconfigs, setConfigFlag, []string{}, "Specify config key-value pairs.")
	planCmd.Flags().IntVar(&flags.progressServerPort, planProgressPortFlag, 0, "Port for the plan progress server. If not provided, the server won't be started.")
	planCmd.Flags().BoolVar(&flags.disableLocalExecution, common.DisableLocalExecutionFlag, false, "Allow files to be executed locally.")
	planCmd.Flags().BoolVar(&flags.failOnEmptyPlan, common.FailOnEmptyPlan, false, "If true, planning will exit with a failure exit code if no services are detected (and no default transformers are found).")

	must(planCmd.Flags().MarkHidden(planProgressPortFlag))

	return planCmd
}