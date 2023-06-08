package main

import (
	"crypto/md5"
	"encoding/hex"
	log "github.com/cihub/seelog"
	"github.com/jordan-wright/email"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"time"
)

func init() {
	// 等系统启动完成  否则时间不准  休息10秒
	time.Sleep(10 * time.Second)
	logger, err := log.LoggerFromConfigAsFile("seelog.xml")
	if err != nil {
		return
	}
	_ = log.ReplaceLogger(logger)
	log.Info("项目启动")
}
func main() {
	log.Debug("进入main方法")
	for {
		log.Debug("开始休息30分")
		time.Sleep(30 * time.Minute)
		log.Debug("结束休息30分")
		checkExpired()
		checkUpdate()
	}
}

func checkExpired() {
	// 判断文件是否存在
	var limitDateFile = "/mofahezi/limit_date.txt"
	_, err := os.Stat(limitDateFile)
	nowStr := getNowStr()
	if err == nil {
		log.Debug("过期文件存在")
		readFile, _ := os.ReadFile(limitDateFile)
		limitDateStr := string(readFile)
		log.Debug("过期时间为:" + limitDateStr)
		log.Debug("现在时间为:" + nowStr)
		if limitDateStr < nowStr {
			log.Debug("已经过期")
			log.Debug("执行关机")
			hostname, _ := os.Hostname()
			sendMail("魔法盒子已过期", "主机名称:"+hostname+",过期时间为:"+limitDateStr+",现在时间为:"+nowStr)
			cmd := exec.Command("halt")
			err := cmd.Run()
			if err != nil {
				_ = log.Error(err)
			}
		} else {
			log.Debug("没有过期")
		}
	}
	if os.IsNotExist(err) {
		log.Debug("过期文件不存在")
		_, _ = os.OpenFile(limitDateFile, os.O_WRONLY, 0666)
		defaultExpireDate := getDefaultExpireStr()
		log.Debug("写入默认时间:" + defaultExpireDate)
		err := os.WriteFile(limitDateFile, []byte(defaultExpireDate), 0666)
		if err != nil {
			_ = log.Error(err)
		}
	}
}

func getNowStr() string {
	const LAYOUT = "2006-01-02 15:04:05"
	// 获取当前日期
	now := time.Now()
	ret := now.Format(LAYOUT)
	return ret
}

func getDefaultExpireStr() string {
	const LAYOUT = "2006-01-02 15:04:05"
	// 获取当前日期
	defaultTime := time.Now().Add(24 * (365 + 15) * time.Hour)
	ret := defaultTime.Format(LAYOUT)
	return ret
}

func checkUpdate() {
	log.Debug("开始检测更新")
	//1. 下载校验文件
	//		如果没有特殊的校验文件,则下载公共的校验文件
	//	根据校验文件判断要不要更新
	// 如果要更新,则下载更新文件
	md5Url := "https://www.mofahezi.net/api/getUpdateMD5?deviceName=" + getHostName()
	upgradeFileUrl := "https://www.mofahezi.net/api/getUpdateFile?deviceName=" + getHostName()
	upgradeFilepath := "/mofahezi/upgrade.tar.gz"
	// 创建 HTTP 请求
	md5Resp, err := http.Get(md5Url)
	if err != nil || md5Resp.StatusCode != http.StatusOK {

	}
	defer md5Resp.Body.Close()
	bodyTmp, err := io.ReadAll(md5Resp.Body) //把响应的body读出
	if err != nil {                          //如果有异常
		log.Debug(err)
	}
	onlineMd5String := string(bodyTmp)
	localUpgradeMd5, _ := FileMD5(upgradeFilepath)
	if onlineMd5String == localUpgradeMd5 {
		log.Debug("MD5值一致，不需要升级，值为：" + onlineMd5String)
	} else {
		log.Debug("MD5值不一致，需要升级，在线MD5值为：" + onlineMd5String + "，本地MD5值为：" + localUpgradeMd5)
		upgradeFile, _ := os.Create(upgradeFilepath)
		upgradeResp, _ := http.Get(upgradeFileUrl)
		io.Copy(upgradeFile, upgradeResp.Body)
		defer upgradeFile.Close()
		// 校验新下载的升级文件
		fileMd5, _ := FileMD5(upgradeFilepath)
		if onlineMd5String == fileMd5 {
			// 升级文件校验成功
			upgradeCmd := exec.Command("/sbin/sysupgrade", "--restore-backup", upgradeFilepath)
			upgradeErr := upgradeCmd.Run()
			if upgradeErr != nil {
				log.Info("升级命令执行失败")
				log.Info(upgradeErr)
				return
			}
			log.Info("升级成功(执行重启之前),md5为:" + fileMd5)
			sendMail("升级成功", "设备名称:"+getHostName()+",升级包的md5值为:"+fileMd5+"，老的升级包的md5值为:"+localUpgradeMd5)
			rebootCmd := exec.Command("reboot")
			rebootErr := rebootCmd.Run()
			if rebootErr != nil {
				log.Info("重启命令执行失败")
				log.Info(rebootErr)
				return
			}
			log.Info("升级成功(执行重启之后),md5为:" + fileMd5)
		} else {
			log.Info("文件校验失败，在线MD5值为" + onlineMd5String + "，新下载的文件的MD5值为：" + fileMd5)
		}
	}
}

func sendMail(subject string, content string) {
	e := email.NewEmail()
	e.From = "xianqielu869@163.com"
	e.To = []string{"mofahezi@gmail.com", "miao.zilong@outlook.com"}
	e.Subject = subject
	e.Text = []byte(content)
	err2 := e.Send("smtp.163.com:25", smtp.PlainAuth("",
		"xianqielu869@163.com",
		"QNVZHBJRPFRQWBSI",
		"smtp.163.com"))
	if err2 != nil {
		log.Debug(err2)
		_ = log.Error("发送失败")
	}
}

func getHostName() string {
	name, _ := os.Hostname()
	return name
}

func FileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), nil
}
