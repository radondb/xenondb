package main

import (
	"fmt"

	"github.com/spf13/cobra"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/radondb/radondb-mysql-kubernetes/sidecar"
	"github.com/radondb/radondb-mysql-kubernetes/utils"
)

const (
	SidecarName  = "sidecar"                             // The name of the sidecar.
	SidecarShort = "A simple helper for mysql operator." // The short description of the sidecar.
)

var (
	log = logf.Log.WithName("sidecar")
	// A command for sidecar.
	cmd = &cobra.Command{
		Use:   SidecarName,
		Short: SidecarShort,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("run the sidecar, see help section")
		},
	}
)

func init() {
	// setup logging
	logf.SetLogger(zap.New(zap.UseDevMode(true)))
}

func main() {
	containerName := sidecar.GetContainerType()
	stop := make(chan struct{}, 1)

	switch containerName {
	case utils.ContainerBackupName:
		backupCfg := sidecar.NewBackupConfig()
		httpCmd := &cobra.Command{
			Use:   "http",
			Short: "start http server",
			Run: func(cmd *cobra.Command, args []string) {
				if err := sidecar.RunHttpServer(backupCfg, stop); err != nil {
					log.Error(err, "run command failed")
				}
			},
		}
		cmd.AddCommand(httpCmd)

	case utils.ContainerBackupJobName:
		reqBackupCfg := sidecar.NewReqBackupConfig()
		reqBackupCmd := &cobra.Command{
			Use:   "request_a_backup",
			Short: "start request a backup",
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) != 1 {
					return fmt.Errorf("require one arguments. ")
				}
				return nil
			},
			Run: func(cmd *cobra.Command, args []string) {
				if err := sidecar.RunRequestBackup(reqBackupCfg, args[0]); err != nil {
					log.Error(err, "run command failed")
				}
			},
		}
		cmd.AddCommand(reqBackupCmd)

	default:
		initCfg := sidecar.NewInitConfig()
		initCmd := sidecar.NewInitCommand(initCfg)
		cmd.AddCommand(initCmd)
	}

	if err := cmd.Execute(); err != nil {
		log.Error(err, "failed to execute command", "cmd", cmd)
	}
}
