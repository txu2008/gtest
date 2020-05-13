package cmd

import (
	"fmt"

	"pzatest/libs/runner/stress"
	"pzatest/models"
	"pzatest/vizion/testcase"

	"github.com/spf13/cobra"
)

var s3TestConf = models.S3TestInput{}

var s3TestCaseArray = map[string]string{
	"upload":          "s3 upload test",
	"download":        "s3 download test:TODO",
	"upload_download": "s3 upload/download test (default)",
}

// s3Cmd represents the s3 command
var s3Cmd = &cobra.Command{
	Use:   "s3",
	Short: "Vizion S3 IO Stress",
	Long:  fmt.Sprintf(`Vizion S3 upload/download files.%s`, CaseMapToString(s3TestCaseArray)),
	Run: func(cmd *cobra.Command, args []string) {
		if len(caseList) == 0 {
			caseList = []string{"upload_download"}
		}
		logger.Infof("Case List(s3): %s", caseList)
		testJobs := []stress.Job{}
		for _, tc := range caseList {
			jobs := []stress.Job{}
			switch tc {
			case "upload":
				upload := func() error {
					_, err := testcase.MultiS3UploadFiles(s3TestConf)
					return err
				}
				jobs = []stress.Job{
					{
						Fn:       upload,
						Name:     "S3 Upload",
						RunTimes: runTimes,
					},
				}
			case "upload_download":
				updownload := func() error {
					return testcase.MultiS3UploadDownloadListDeleteFiles(s3TestConf)
				}
				jobs = []stress.Job{
					{
						Fn:       updownload,
						Name:     "S3 Upload/List/Download/Delete",
						RunTimes: runTimes,
					},
				}
			}
			testJobs = append(testJobs, jobs...)
		}
		stress.Run(testJobs)
	},
}

func init() {
	stressCmd.AddCommand(s3Cmd)
	suite = "s3"
	s3Cmd.PersistentFlags().StringVar(&s3TestConf.S3Ip, "s3_ip", "", "S3 server IP address")
	s3Cmd.PersistentFlags().StringVar(&s3TestConf.S3AccessID, "s3_access_id", "", "S3 access ID")
	s3Cmd.PersistentFlags().StringVar(&s3TestConf.S3SecretKey, "s3_secret_key", "", "S3 access secret key")
	s3Cmd.PersistentFlags().IntVar(&s3TestConf.S3Port, "s3_port", 443, "S3 server access port")
	s3Cmd.PersistentFlags().StringVar(&s3TestConf.S3Bucket, "s3_bucket", "", "S3 bucket name")
	s3Cmd.PersistentFlags().StringVar(&s3TestConf.LocalDataDir, "local_dir", "/tmp/", "Local dir for store S3 test files")
	s3Cmd.PersistentFlags().IntVar(&s3TestConf.RandomPercent, "random_percent", 100, "Percent of S3 test files with random data")
	s3Cmd.PersistentFlags().IntVar(&s3TestConf.EmptyPercent, "empty_percent", 0, "Percent of S3 test files with empty data")
	s3Cmd.PersistentFlags().BoolVar(&s3TestConf.RenameFile, "rename", false, "Rename files name each time if true")
	s3Cmd.PersistentFlags().BoolVar(&s3TestConf.DeleteFile, "delete", false, "Delete files from s3 bucket after test if true")
	s3Cmd.PersistentFlags().IntVar(&s3TestConf.Clients, "client", 1, "S3 Client number for test at the same time")
	s3Cmd.PersistentFlags().StringArrayVar(&s3TestConf.FileInputs, "files", []string{"txt:20:1k-10k", "dd:1:100mb"}, "S3 files config array")

	s3Cmd.MarkPersistentFlagRequired("s3_ip")
	s3Cmd.MarkPersistentFlagRequired("s3_bucket")
}
