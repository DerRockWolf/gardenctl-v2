/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardenctl-v2/internal/util"
	cmdssh "github.com/gardener/gardenctl-v2/pkg/cmd/ssh"
	cmdtarget "github.com/gardener/gardenctl-v2/pkg/cmd/target"
	cmdversion "github.com/gardener/gardenctl-v2/pkg/cmd/version"
	"github.com/gardener/gardenctl-v2/pkg/target"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

const (
	envPrefix        = "GCTL"
	envGardenHomeDir = envPrefix + "_HOME"
	envConfigName    = envPrefix + "_CONFIG_NAME"

	gardenHomeFolder = ".garden"
	configName       = "gardenctl-v2"
	targetFilename   = "target.yaml"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the root cmd.
func Execute() {
	cmd := NewDefaultGardenctlCommand()
	// any error would already be printed, so avoid doing it again here
	if cmd.Execute() != nil {
		os.Exit(1)
	}
}

// NewDefaultGardenctlCommand creates the `gardenctl` command with defaults
func NewDefaultGardenctlCommand() *cobra.Command {
	factory := util.FactoryImpl{
		TargetFlags: target.NewTargetFlags("", "", "", ""),
	}
	ioStreams := util.NewIOStreams()

	return NewGardenctlCommand(&factory, ioStreams)
}

// NewGardenctlCommand creates the `gardenctl` command
func NewGardenctlCommand(f *util.FactoryImpl, ioStreams util.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gardenctl",
		Short:        "gardenctl is a utility to interact with Gardener installations",
		SilenceUsage: true,
	}

	cmd.SetIn(ioStreams.In)
	cmd.SetOut(ioStreams.Out)
	cmd.SetErr(ioStreams.ErrOut)

	// register initializers
	cobra.OnInitialize(func() {
		initConfig(f)
	})

	flags := cmd.PersistentFlags()
	// Do not precalculate what $HOME is for the help text, because it prevents
	// usage where the current user has no home directory (which might _just_ be
	// the reason the user chose to specify an explicit config file).
	flags.StringVar(&f.ConfigFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s/%s.yaml)", gardenHomeFolder, configName))

	// allow to temporarily re-target a different cluster
	f.TargetFlags.AddFlags(flags)

	registerCompletionFuncForGlobalFlags(cmd, f)

	// add subcommands
	cmd.AddCommand(cmdssh.NewCmdSSH(f, cmdssh.NewSSHOptions(ioStreams)))
	cmd.AddCommand(cmdtarget.NewCmdTarget(f, cmdtarget.NewTargetOptions(ioStreams)))
	cmd.AddCommand(cmdversion.NewCmdVersion(f, cmdversion.NewVersionOptions(ioStreams)))

	return cmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig(f *util.FactoryImpl) {
	var err error

	if f.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(f.ConfigFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		configPath := filepath.Join(home, gardenHomeFolder)

		// Search config in $HOME/.garden or in path provided with the env variable GCTL_HOME with name ".garden-login" (without extension) or name from env variable GCTL_CONFIG_NAME.
		envHomeDir, err := homedir.Expand(os.Getenv(envGardenHomeDir))
		cobra.CheckErr(err)

		viper.AddConfigPath(envHomeDir)
		viper.AddConfigPath(configPath)
		if os.Getenv(envConfigName) != "" {
			viper.SetConfigName(os.Getenv(envConfigName))
		} else {
			viper.SetConfigName(configName)
		}
	}

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		klog.Errorf("failed to read config file: %v", err)
	}

	// initialize the factory

	// prefer an explicit GCTL_HOME env,
	// but fallback to the system-defined home directory
	home := os.Getenv(envGardenHomeDir)
	if len(home) == 0 {
		home, err = homedir.Dir()
		cobra.CheckErr(err)

		home = filepath.Join(home, gardenHomeFolder)
	}

	f.ConfigFile = viper.ConfigFileUsed()
	f.GardenHomeDirectory = home
	targetFile := filepath.Join(home, targetFilename)
	f.TargetFile = targetFile
}

func registerCompletionFuncForGlobalFlags(cmd *cobra.Command, f *util.FactoryImpl) {
	utilruntime.Must(cmd.RegisterFlagCompletionFunc("garden", completionWrapper(f, gardenFlagCompletionFunc)))
	utilruntime.Must(cmd.RegisterFlagCompletionFunc("project", completionWrapper(f, projectFlagCompletionFunc)))
	utilruntime.Must(cmd.RegisterFlagCompletionFunc("seed", completionWrapper(f, seedFlagCompletionFunc)))
	utilruntime.Must(cmd.RegisterFlagCompletionFunc("shoot", completionWrapper(f, shootFlagCompletionFunc)))
}

type cobraCompletionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
type cobraCompletionFuncWithError func(ctx context.Context, manager target.Manager, tf target.TargetFlags) ([]string, error)

func completionWrapper(f *util.FactoryImpl, completer cobraCompletionFuncWithError) cobraCompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tf := f.TargetFlags

		// By default, the factory will provide a target manager that uses the dynamicTargetProvider (DTP)
		// implementation, i.e. is based on the file just as much as the CLI flags.
		// The DTP tries to allow users to "move up", i.e. when they already targeted a shoot, just adding
		// "--garden foo" should not just change the used garden cluster, but _target_ the garden (instead
		// of the shoot). This behaviour is not suitable for the CLI completion functions, because
		// when completing "gardenctl --garden foo --shoot [tab]", the DTP would consider this as
		// "user wants to target the garden" and will therefore throw away the project/seed information.
		// Project and seed information however are important for the completion functions.
		//
		// To work around this, all completion functions use a manager with a regular filesystem based
		// target provider without considering the given target flags.
		manager, err := f.WithoutTargetFlags().Manager()

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		result, err := completer(f.Context(), manager, tf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return util.FilterStringsByPrefix(toComplete, result), cobra.ShellCompDirectiveNoFileComp
	}
}

func gardenFlagCompletionFunc(ctx context.Context, manager target.Manager, tf target.TargetFlags) ([]string, error) {
	return util.GardenNames(manager)
}

func projectFlagCompletionFunc(ctx context.Context, manager target.Manager, tf target.TargetFlags) ([]string, error) {
	// any --garden flag has precedence over the config file
	var currentTarget target.Target

	if tf.GardenName() != "" {
		currentTarget = target.NewTarget(tf.GardenName(), "", "", "")
	} else {
		var err error

		currentTarget, err = manager.CurrentTarget()
		if err != nil {
			return nil, fmt.Errorf("failed to read current target: %w", err)
		}
	}

	return util.ProjectNamesForTarget(ctx, manager, currentTarget)
}

func seedFlagCompletionFunc(ctx context.Context, manager target.Manager, tf target.TargetFlags) ([]string, error) {
	// any --garden flag has precedence over the config file
	var currentTarget target.Target

	if tf.GardenName() != "" {
		currentTarget = target.NewTarget(tf.GardenName(), "", "", "")
	} else {
		var err error
		currentTarget, err = manager.CurrentTarget()
		if err != nil {
			return nil, fmt.Errorf("failed to read current target: %w", err)
		}
	}

	return util.SeedNamesForTarget(ctx, manager, currentTarget)
}

func shootFlagCompletionFunc(ctx context.Context, manager target.Manager, tf target.TargetFlags) ([]string, error) {
	// errors are okay here, as we patch the target anyway
	currentTarget, _ := manager.CurrentTarget()

	if tf.GardenName() != "" {
		currentTarget = currentTarget.WithGardenName(tf.GardenName())
	}

	if tf.ProjectName() != "" {
		currentTarget = currentTarget.WithProjectName(tf.ProjectName()).WithSeedName("")
	} else if tf.SeedName() != "" {
		currentTarget = currentTarget.WithSeedName(tf.SeedName()).WithProjectName("")
	}

	return util.ShootNamesForTarget(ctx, manager, currentTarget)
}
