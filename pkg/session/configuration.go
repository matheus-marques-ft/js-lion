package session

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"lion/pkg/config"
	"lion/pkg/guacd"

	"github.com/jumpserver-dev/sdk-go/common"
	"github.com/jumpserver-dev/sdk-go/model"
)

type ConnectionConfiguration interface {
	GetGuacdConfiguration() guacd.Configuration
}

var (
	_ ConnectionConfiguration = RDPConfiguration{}
	_ ConnectionConfiguration = VNCConfiguration{}
	_ ConnectionConfiguration = VirtualAppConfiguration{}
)

type RDPConfiguration struct {
	SessionId      string
	Created        common.UTCTime
	User           *model.User
	Asset          *model.Asset
	Account        *model.Account
	Platform       *model.Platform
	TerminalConfig *model.TerminalConfig
	ActionsPerm    *ActionPermission
}

func (r RDPConfiguration) GetGuacdConfiguration() guacd.Configuration {
	var (
		username string
		password string
		ip       string
		port     string
		adDomain string
	)

	ip = r.Asset.Address
	port = strconv.Itoa(r.Asset.ProtocolPort(rdp))
	username = r.Account.Username
	password = r.Account.Secret

	conf := guacd.NewConfiguration()
	conf.Protocol = rdp
	conf.SetParameter(guacd.Hostname, ip)
	conf.SetParameter(guacd.Port, port)

	/*
		 pam will handle the ad Domain info and convert it into the username@domain format
		 no longer processed from platform
			if r.Platform != nil {
				if rdpSetting, ok := r.Platform.GetProtocolSetting(rdp); ok {
					if rdpSetting.Setting.AdDomain != "" {
						adDomain = rdpSetting.Setting.AdDomain
					}
				}
			}

			/*
				AD Domain handling adjusted to:
				1. if the account username is in domain\username format, convert it to username@domain, overriding the platform's AD domain setting.
				2. for other account formats, use the platform's AD domain setting if configured, otherwise leave it unset
	*/

	parts := strings.Split(username, `\`)
	if len(parts) == 2 {
		username = fmt.Sprintf("%s@%s", parts[1], parts[0])
		adDomain = parts[0]
	}

	// try to get the AD domain info from a username in the username@domain format
	//if adDomain == "" && strings.Contains(username, `@`) {
	//	adParts := strings.Split(username, `@`)
	//	if len(adParts) >= 2 {
	//		adDomain = adParts[len(adParts)-1]
	//	}
	//}
	// if domain and username both set the AD domain info at the same time, freerdp connection will fail
	// domain and username cannot both carry the AD domain info at the same time

	conf.SetParameter(guacd.RDPUsername, username)
	conf.SetParameter(guacd.RDPPassword, password)
	if adDomain != "" {
		conf.SetParameter(guacd.RDPDomain, adDomain)
	}

	//// set the recording path
	//if r.TerminalConfig.ReplayStorage.TypeName != "null" {
	//	recordDirPath := filepath.Join(config.GlobalConfig.RecordPath,
	//		r.Created.Format(recordDirTimeFormat))
	//	conf.SetParameter(guacd.RecordingPath, recordDirPath)
	//	conf.SetParameter(guacd.CreateRecordingPath, BoolTrue)
	//	conf.SetParameter(guacd.RecordingName, r.SessionId)
	//}

	// display related
	{
		for key, value := range RDPDisplay.GetDisplayParams() {
			conf.SetParameter(key, value)
		}
		for key, value := range RDPBuiltIn {
			conf.SetParameter(key, value)
		}
		// reconnect would cause multiple recording files to be created
		conf.SetParameter(guacd.RDPResizeMethod, "display-update")
	}

	// set the mounted directory for upload/download
	{
		driverShareId := r.User.ID
		if config.GlobalConfig.DriveScope == config.DriverScopeSession {
			driverShareId = r.SessionId
		}
		drivePath := filepath.Join(config.GlobalConfig.DrivePath, driverShareId)
		enableDrive := ConvertBoolToString(r.ActionsPerm.EnableDownload || r.ActionsPerm.EnableUpload)
		disableDownload := ConvertBoolToString(!r.ActionsPerm.EnableDownload)
		disableUpload := ConvertBoolToString(!r.ActionsPerm.EnableUpload)
		conf.SetParameter(guacd.RDPDrivePath, drivePath)
		conf.SetParameter(guacd.RDPCreateDrivePath, BoolTrue)
		conf.SetParameter(guacd.RDPEnableDrive, enableDrive)
		conf.SetParameter(guacd.RDPDriveName, "Lion")
		conf.SetParameter(guacd.RDPDisableDownload, disableDownload)
		conf.SetParameter(guacd.RDPDisableUpload, disableUpload)
	}

	// copy/paste
	{
		disableCopy := ConvertBoolToString(!r.ActionsPerm.EnableCopy)
		disablePaste := ConvertBoolToString(!r.ActionsPerm.EnablePaste)
		conf.SetParameter(guacd.DisableCopy, disableCopy)
		conf.SetParameter(guacd.DisablePaste, disablePaste)
	}

	// setting from the platform
	rdpSecurityValue := SecurityAny
	if r.Platform != nil {
		if rdpSettings, ok := r.Platform.GetProtocolSetting(rdp); ok {
			setObj := rdpSettings.GetSetting()
			if setObj.Security != "" {
				rdpSecurityValue = setObj.Security
			}
			if setObj.Console {
				conf.SetParameter(guacd.RDPConsole, BoolTrue)
			}
		}
	}
	conf.SetParameter(guacd.RDPSecurity, rdpSecurityValue)
	conf.SetParameter(guacd.RDPIgnoreCert, BoolTrue)

	// set the client name, shown in Task Manager under Users -- Client Name
	conf.SetParameter(guacd.RDPClientName, "Lion")

	return conf
}

type VNCConfiguration struct {
	SessionId      string
	Created        common.UTCTime
	User           *model.User
	Asset          *model.Asset
	Account        *model.Account
	Platform       *model.Platform
	TerminalConfig *model.TerminalConfig
	ActionsPerm    *ActionPermission
}

const recordDirTimeFormat = "2006-01-02"

const nullUsername = "null"

func (r VNCConfiguration) GetGuacdConfiguration() guacd.Configuration {
	conf := guacd.NewConfiguration()
	var (
		username string
		password string
		ip       string
		port     string
	)
	ip = r.Asset.Address
	port = strconv.Itoa(r.Asset.ProtocolPort("vnc"))
	username = r.Account.Username
	password = r.Account.Secret
	if username == nullUsername {
		username = ""
	}
	conf.Protocol = vnc
	conf.SetParameter(guacd.Hostname, ip)
	conf.SetParameter(guacd.Port, port)

	{
		conf.SetParameter(guacd.VNCUsername, username)
		conf.SetParameter(guacd.VNCPassword, password)
		conf.SetParameter(guacd.VNCAutoretry, "3")
	}
	// set storage
	//replayCfg := r.TerminalConfig.ReplayStorage
	//if replayCfg.TypeName != "null" {
	//	recordDirPath := filepath.Join(config.GlobalConfig.RecordPath, r.Created.Format(recordDirTimeFormat))
	//	conf.SetParameter(guacd.RecordingPath, recordDirPath)
	//	conf.SetParameter(guacd.CreateRecordingPath, BoolTrue)
	//	conf.SetParameter(guacd.RecordingName, r.SessionId)
	//}
	{
		for key, value := range VNCDisplay.GetDisplayParams() {
			conf.SetParameter(key, value)
		}
	}

	// copy/paste
	{
		disableCopy := ConvertBoolToString(!r.ActionsPerm.EnableCopy)
		disablePaste := ConvertBoolToString(!r.ActionsPerm.EnablePaste)
		conf.SetParameter(guacd.DisableCopy, disableCopy)
		conf.SetParameter(guacd.DisablePaste, disablePaste)
	}

	// VNC_CLIPBOARD_ENCODING from env
	if value := viper.GetString("VNC_CLIPBOARD_ENCODING"); value != "" {
		conf.SetParameter(guacd.VNCClipboardEncoding, value)
	}

	return conf
}

const (
	BoolFalse = "false"
	BoolTrue  = "true"
)

func ConvertBoolToString(b bool) string {
	if b {
		return BoolTrue
	}
	return BoolFalse
}

type VirtualAppConfiguration struct {
	SessionId      string
	Created        common.UTCTime
	User           *model.User
	VirtualAppOpt  *model.VirtualAppContainer
	TerminalConfig *model.TerminalConfig
	ActionsPerm    *ActionPermission
}

func (r VirtualAppConfiguration) GetGuacdConfiguration() guacd.Configuration {
	conf := guacd.NewConfiguration()
	var (
		username string
		password string
		ip       string
		port     string
	)
	ip = r.VirtualAppOpt.Host
	port = strconv.Itoa(r.VirtualAppOpt.Port)
	username = r.VirtualAppOpt.Username
	password = r.VirtualAppOpt.Password
	sftpPort := r.VirtualAppOpt.SFTPPort
	conf.Protocol = vnc
	conf.SetParameter(guacd.Hostname, ip)
	conf.SetParameter(guacd.Port, port)

	{
		conf.SetParameter(guacd.VNCUsername, username)
		conf.SetParameter(guacd.VNCPassword, password)
		conf.SetParameter(guacd.VNCAutoretry, "10")
	}
	// set storage
	//replayCfg := r.TerminalConfig.ReplayStorage
	//if replayCfg.TypeName != "null" {
	//	recordDirPath := filepath.Join(config.GlobalConfig.RecordPath, r.Created.Format(recordDirTimeFormat))
	//	conf.SetParameter(guacd.RecordingPath, recordDirPath)
	//	conf.SetParameter(guacd.CreateRecordingPath, BoolTrue)
	//	conf.SetParameter(guacd.RecordingName, r.SessionId)
	//}
	{
		for key, value := range VNCDisplay.GetDisplayParams() {
			conf.SetParameter(key, value)
		}
	}

	// copy/paste
	{
		disableCopy := ConvertBoolToString(!r.ActionsPerm.EnableCopy)
		disablePaste := ConvertBoolToString(!r.ActionsPerm.EnablePaste)
		conf.SetParameter(guacd.DisableCopy, disableCopy)
		conf.SetParameter(guacd.DisablePaste, disablePaste)
	}
	// vnc forces the use of utf8 encoding
	conf.SetParameter(guacd.VNCClipboardEncoding, "UTF-8")

	if sftpPort > 0 {
		//  sftp enable and set sftp username and password
		enableDrive := ConvertBoolToString(r.ActionsPerm.EnableDownload || r.ActionsPerm.EnableUpload)
		disableDownload := ConvertBoolToString(!r.ActionsPerm.EnableDownload)
		disableUpload := ConvertBoolToString(!r.ActionsPerm.EnableUpload)
		conf.SetParameter(guacd.EnableSftp, enableDrive)
		conf.SetParameter(guacd.SftpHostname, ip)
		conf.SetParameter(guacd.SftpPort, strconv.Itoa(sftpPort))
		conf.SetParameter(guacd.SftpUsername, vAPPSFTPUsername)
		conf.SetParameter(guacd.SftpPassword, password)
		conf.SetParameter(guacd.SftpRootDirectory, sftpRootDir)
		conf.SetParameter(guacd.SftpDisableDownload, disableDownload)
		conf.SetParameter(guacd.SftpDisableUpload, disableUpload)
	}
	return conf
}

const (
	vAPPSFTPUsername = "jumpserver"
	sftpRootDir      = "/tmp/jumpserver/download"
)
