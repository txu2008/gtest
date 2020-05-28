package cmd

import (
	"fmt"
	"pzatest/libs/runner/stress"
	"pzatest/vizion/maintenance"

	"github.com/spf13/cobra"
)

var maintConf maintenance.MaintTestInput

// maintCmd represents the maint command
var maintCmd = &cobra.Command{
	Use:   "maint",
	Short: "Maintaince mode tools",
	Long: `Maintaince mode operations, subCommand:
	stop/start/restart: stop/start/restart specified services
	cleanup: cleanup specified options: log/journal/etcd ...
	upgrade: upgrade env to specified image
	rolling_update: rolling update env services`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("maint called")
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Maintaince mode tools: stop service",
	Long:    "Stop specified services(default:All DPL+APP)",
	Example: "pzatest maint stop --master_ips 10.25.119.71 --vset_ids 1 --services es --clean all",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Infof("maint stop services ...")
		var maintainer maintenance.Maintainer
		maintainer = maintenance.NewMaint(vizionBaseConf, maintConf)
		stop := func() error {
			err := maintainer.Stop()
			return err
		}
		jobs := []stress.Job{
			{
				Fn:       stop,
				Name:     "Stop-Service",
				RunTimes: 1,
			},
		}
		stress.Run(jobs)
	},
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Maintaince mode tools: start",
	Long:  `start specified services`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("maint start called")
	},
}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Maintaince mode tools: restart",
	Long:  `restart specified services`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("maint restart called")
	},
}

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Maintaince mode tools: cleanup",
	Long:  `cleanup specified items`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("maint cleanup called")
	},
}

// cleanupCmd represents the make_binary command
var makeBinaryCmd = &cobra.Command{
	Use:   "make_binary",
	Short: "Maintaince mode tools: make_binary",
	Long:  `make binary from git server`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("maint make_binary called")
	},
}

// AddFlagsMaintService ...
func AddFlagsMaintService(cmd *cobra.Command) {
	cmd.PersistentFlags().StringArrayVar(&maintConf.SvNameArr, "services", []string{}, "Service Name List")
	cmd.PersistentFlags().StringArrayVar(&maintConf.ExculdeSvNameArr, "services_exclude", []string{}, "Service Name List which excluded")
}

// AddFlagsMaintClean ...
func AddFlagsMaintClean(cmd *cobra.Command) {
	cmd.PersistentFlags().StringArrayVar(&maintConf.CleanNameArr, "clean", []string{}, "Clean item Name List")
}

func init() {
	rootCmd.AddCommand(maintCmd)
	maintCmd.AddCommand(stopCmd)
	maintCmd.AddCommand(startCmd)
	maintCmd.AddCommand(restartCmd)
	maintCmd.AddCommand(cleanupCmd)
	maintCmd.AddCommand(makeBinaryCmd)

	AddFlagsMaintService(stopCmd)
	AddFlagsMaintClean(stopCmd)
	AddFlagsMaintService(startCmd)
	AddFlagsMaintService(restartCmd)

}
