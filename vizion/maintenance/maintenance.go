package maintenance

import (
	"errors"
	"fmt"
	"os"
	"path"
	"pzatest/config"
	"pzatest/libs/git"
	"pzatest/types"
	"pzatest/vizion/resources"
	"strings"
	"time"

	"github.com/chenhg5/collection"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("test")

// Maintainer for maintenance ops
type Maintainer interface {
	Cleanup() error
	Stop() error
	Start() error
	Restart() error
	MakeBinary() error
	MakeImage() error
	ApplyImage() error
	UpgradeCore() error
}

// Maint is used to interact with features provided by the  group.
type Maint struct {
	Vizion            resources.Vizion
	ServiceArr        []config.Service
	ExculdeServiceArr []config.Service
	BinaryArr         []config.Service
	CleanArr          []config.CleanItem
	Image             string
	GitCfg            GitInput
}

// GitInput ...
type GitInput struct {
	BuildServerIP   string // the git project server IP address
	BuildServerUser string
	BuildServerPwd  string
	BuildServerKey  string
	BuildPath       string // the git procject path
	BuildNum        string // the build number for tag, eg: 2.1.0.133, used in JENKINS
	Pull            bool   // git pull before tag?
	Tag             bool   // git tag && push  ?
	Make            bool   // exec make file ?
	LocalBinPath    string // local path for store dpl binarys
}

// MaintTestInput .
type MaintTestInput struct {
	SvNameArr         []string // service Name array
	ExculdeSvNameArr  []string // service Name array
	BinNameArr        []string //  binary Name array
	ExculdeBinNameArr []string //  binary Name array
	CleanNameArr      []string //  clean item Name array
	Image             string   // eg: registry.ai/stable:tag
	GitCfg            GitInput // The build number for image tag name, used in JENKINS
}

// NewMaint returns a Nodes
func NewMaint(base types.VizionBaseInput, mt MaintTestInput) *Maint {
	var svArr, binArr []config.Service
	var cleanArr []config.CleanItem

	// service Array
	if len(mt.SvNameArr) == 0 {
		svArr = config.DefaultCoreServiceArray
	} else {
		for _, item := range config.DefaultCoreServiceArray {
			if collection.Collect(mt.SvNameArr).Contains(item.Name) {
				svArr = append(svArr, item)
			}
		}
	}

	// binary Array
	if len(mt.BinNameArr) == 0 {
		binArr = svArr
	} else {
		for _, item := range config.DefaultDplBinaryArray {
			if collection.Collect(mt.BinNameArr).Contains(item.Name) {
				binArr = append(binArr, item)
			}
		}
	}

	// clean Array
	if len(mt.CleanNameArr) == 0 {
		cleanArr = []config.CleanItem{}
	} else if collection.Collect(mt.CleanNameArr).Contains("all") {
		cleanArr = config.DefaultCleanArray
	} else {
		for _, item := range config.DefaultCleanArray {
			if collection.Collect(mt.CleanNameArr).Contains(item.Name) {
				cleanArr = append(cleanArr, item)
			}
		}
	}

	return &Maint{
		Vizion:     resources.Vizion{Base: base},
		ServiceArr: svArr,
		BinaryArr:  binArr,
		CleanArr:   cleanArr,
		Image:      mt.Image,
		GitCfg:     mt.GitCfg,
	}
}

// Cleanup - maint
func (maint *Maint) Cleanup() error {
	var err error
	formatBD := false
	for _, clean := range maint.CleanArr {
		switch clean.Name {
		case "log":
			err = maint.Vizion.CleanLog(maint.ServiceArr)
			if err != nil {
				return err
			}
		case "journal":
			err = maint.Vizion.CleanEtcd(clean.Arg)
			if err != nil {
				return err
			}
			err = maint.Vizion.CleanJournal()
			if err != nil {
				return err
			}
		case "storage_cache":
			err = maint.Vizion.CleanStorageCache(clean.Arg[0], false)
			if err != nil {
				return err
			}
		case "master_cassandra":
			formatBD = true
			err = maint.Vizion.UpdateMasterCassTables()
			if err != nil {
				return err
			}
		case "sub_cassandra":
			formatBD = true
			err = maint.Vizion.CleanSubCassTables(clean.Arg)
			if err != nil {
				return err
			}
		case "etcd":
			formatBD = true
			err = maint.Vizion.CleanEtcd(clean.Arg)
			if err != nil {
				return err
			}
		case "cdcgc":
			err = maint.Vizion.CleanCdcgc()
			if err != nil {
				return err
			}
		}
	}

	if formatBD == true {
		err = maint.Vizion.UpdateSubCassTables()
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop - maint
func (maint *Maint) Stop() error {
	var err error
	stopServiceArr := config.ReverseServiceArr(maint.ServiceArr)
	err = maint.Vizion.StopServices(stopServiceArr)
	if err != nil {
		return err
	}

	err = maint.Cleanup()
	if err != nil {
		return err
	}

	return nil
}

// Start - maint
func (maint *Maint) Start() error {
	// logger.Info(utils.Prettify(maint))
	err := maint.Vizion.StartServices(maint.ServiceArr)
	return err
}

// Restart - maint
func (maint *Maint) Restart() error {
	var err error
	err = maint.Stop()
	if err != nil {
		return err
	}
	err = maint.Start()
	if err != nil {
		return err
	}

	return nil
}

// MakeBinary - maint
func (maint *Maint) MakeBinary() error {
	var err error
	strTime := time.Now().Format("2006-01-02-15-04-05")
	// Get build path branch Name, joint tagName
	gitMgr := git.NewGitMgr(
		maint.GitCfg.BuildServerIP,
		maint.GitCfg.BuildServerUser,
		maint.GitCfg.BuildServerPwd,
		maint.GitCfg.BuildServerKey,
	)
	branchName := gitMgr.GetCurrentBranch(maint.GitCfg.BuildPath)
	tagName := strTime + "-" + branchName
	if maint.GitCfg.BuildNum == "" {
		tagName = tagName + "_private"
	} else {
		tagName = tagName + "_" + maint.GitCfg.BuildNum
	}

	localBinPath := path.Join(maint.GitCfg.LocalBinPath, tagName)
	err = os.MkdirAll(localBinPath, os.ModePerm)
	if err != nil {
		logger.Panic(err)
	}

	// pull && save changelog
	if maint.GitCfg.Pull == true {
		if err = gitMgr.Pull(maint.GitCfg.BuildPath); err != nil {
			return err
		}
		// change.log
		date, changeLog := gitMgr.GetChangeLog(maint.GitCfg.BuildPath)
		changeLogFile := path.Join(localBinPath, "change.log")
		logger.Info(changeLogFile)
		file, err := os.OpenFile(changeLogFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			logger.Panic(err)
		}
		defer file.Close()
		file.WriteString(date + "\n")
		file.WriteString("Version:" + tagName + "\n")
		file.WriteString("Change logs:\n" + changeLog)
	}

	// make file
	if maint.GitCfg.Make == true {
		for _, bin := range maint.BinaryArr {
			binGitPath := path.Join(maint.GitCfg.BuildPath, bin.GitPath)
			binName := bin.Name
			binGitPathName := path.Join(binGitPath, binName)
			binLocalPathName := path.Join(localBinPath, binName)
			if md5sum := gitMgr.MakeFile(binGitPath, binName); md5sum == "" {
				return fmt.Errorf("%s make failed", binName)
			}
			gitMgr.ConnectSftpClient()
			if err = gitMgr.ScpGet(binLocalPathName, binGitPathName); err != nil {
				return err
			}
		}
	}
	logger.Infof("Local Binary Path: %s", localBinPath)
	return nil
}

// MakeImage - maint make image by tag to gitlab
func (maint *Maint) isImageOK() error {
	logger.Infof("Wait for Image Availabel: %s", maint.Image)
	if maint.Image == "" {
		return errors.New("image name is nul")
	}
	tagName := strings.Split(maint.Image, ":")[1]

	// wait for image OK on gitlab
	cfg := git.GitlabConfig{
		BaseURL: "http://gitlab.panzura.com",
		Token:   "xjB1FHHyJHNQUhgy7K7t",
	}
	projectID := 25
	gitlabMgr := git.NewGitlabClient(cfg)
	err := gitlabMgr.IsPipelineJobsSuccess(projectID, tagName)
	if err != nil {
		return err
	}
	logger.Infof("Image Availabel: %s", maint.Image)
	return nil
}

// MakeImage - maint make image by tag to gitlab
func (maint *Maint) MakeImage() error {
	var err error
	strTime := time.Now().Format("2006-01-02-15-04-05")
	// Get build path branch Name, joint tagName
	gitMgr := git.NewGitMgr(
		maint.GitCfg.BuildServerIP,
		maint.GitCfg.BuildServerUser,
		maint.GitCfg.BuildServerPwd,
		maint.GitCfg.BuildServerKey,
	)
	branchName := gitMgr.GetCurrentBranch(maint.GitCfg.BuildPath)
	tagName := strTime + "-" + branchName
	if maint.GitCfg.BuildNum == "" {
		tagName = tagName + "_notest"
	} else {
		tagName = tagName + "_" + maint.GitCfg.BuildNum
	}

	// pull && save changelog
	if maint.GitCfg.Pull == true {
		if err = gitMgr.Pull(maint.GitCfg.BuildPath); err != nil {
			return err
		}
		// change.log
		date, changeLog := gitMgr.GetChangeLog(maint.GitCfg.BuildPath)
		localBinPath := path.Join(maint.GitCfg.LocalBinPath, tagName)
		err := os.MkdirAll(localBinPath, os.ModePerm)
		if err != nil {
			logger.Panic(err)
		}
		changeLogFile := path.Join(localBinPath, "change.log")
		logger.Info(changeLogFile)
		file, err := os.OpenFile(changeLogFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			logger.Panic(err)
		}
		defer file.Close()
		file.WriteString(date + "\n")
		file.WriteString("Version/Tag:" + tagName + "\n")
		file.WriteString("Change logs:\n" + changeLog)
	}

	// tag && push
	if maint.GitCfg.Tag == true {
		if err = gitMgr.Tag(maint.GitCfg.BuildPath, tagName); err != nil {
			return err
		}
	}

	maint.Image = config.RemoteDplRegistry + ":" + tagName

	// wait for maint.Image OK on gitlab
	if err = maint.isImageOK(); err != nil {
		return err
	}
	// logger.Info(utils.Prettify(maint))
	return nil
}

// ApplyImage - maint TODO
func (maint *Maint) ApplyImage() error {
	var err error
	// wait for image OK on gitlab
	if err = maint.isImageOK(); err != nil {
		return err
	}

	err = maint.Stop()
	if err != nil {
		return err
	}

	err = maint.Vizion.ApplyServicesImage(maint.ServiceArr, maint.Image)
	if err != nil {
		return err
	}

	err = maint.Vizion.ApplyDplmanagerShellImage(maint.Image)
	if err != nil {
		return err
	}

	err = maint.Start()
	if err != nil {
		return err
	}

	return nil
}

// UpgradeCore - maint
func (maint *Maint) UpgradeCore() error {
	var err error
	err = maint.MakeImage()
	if err != nil {
		return err
	}

	err = maint.Stop()
	if err != nil {
		return err
	}

	err = maint.ApplyImage()
	if err != nil {
		return err
	}

	return nil
}
