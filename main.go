package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

func init() {
	logger, err := log.LoggerFromConfigAsFile("seelog.xml")
	if err != nil {
		return
	}
	_ = log.ReplaceLogger(logger)
	log.Info("项目启动")
}
func main() {
	log.Debug("程序启动")
	//err := Dir(
	//	"C:\\Users\\Administrator\\GolandProjects\\mofahezi-api\\update_package\\device00000",
	//	"C:\\Users\\Administrator\\GolandProjects\\mofahezi-api\\update_package\\device00005")
	//fmt.Println(err) // nil

	var err error
	var dirs []os.DirEntry
	var fileMD5 string
	if dirs, err = os.ReadDir("./update_package"); err != nil {
		log.Debug("读取目录失败")
		return
	}
	for _, v := range dirs {
		if v.IsDir() {
			if v.Name() != "device00000" {
				Dir("update_package/device00000", "update_package/"+v.Name())
			}
			gzFilePath := "update_package/" + v.Name() + ".tar.gz"
			createTarGz("update_package/"+v.Name(), gzFilePath)
			if fileMD5, err = FileMD5(gzFilePath); err != nil {
				log.Debug("生成MD5失败")
			}
			os.WriteFile("update_package/"+v.Name()+".md5", []byte(fileMD5), 0666)
		}
	}

	http.HandleFunc("/getUpdateMD5", GetUpdateMD5)
	http.HandleFunc("/getUpdateFile", GetUpdateFile)
	http.ListenAndServe(":8080", nil)
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

func Dir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = Dir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = File(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}
func File(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if _, err = os.Stat(dst); !(err != nil && os.IsNotExist(err)) {
		// 文件已经存在
		return nil
	}

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func GetUpdateMD5(w http.ResponseWriter, req *http.Request) {
	log.Debug("GetUpdateMD5开始")
	values := req.URL.Query()
	deviceName := values.Get("deviceName")
	log.Debug("设备名称" + deviceName)
	if deviceName != "" {
		md5, readMd5FileErr := os.ReadFile("./update_package/" + deviceName + ".md5")
		if readMd5FileErr != nil {
			log.Debug("找不到指定设备的升级文件MD5，使用默认文件")
			md5, _ = os.ReadFile("./update_package/" + "device00000" + ".md5")
		}
		w.Write(md5)
	}
}

func GetUpdateFile(rsp http.ResponseWriter, req *http.Request) {
	var err error
	var b []byte
	//获取请求参数
	values := req.URL.Query()
	deviceName := values.Get("deviceName")
	//设置响应头
	header := rsp.Header()
	header.Add("Content-Type", "application/octet-stream")
	fileSuffix := ".tar.gz"
	header.Add("Content-Disposition", "attachment;filename="+deviceName+fileSuffix)
	if b, err = os.ReadFile("./update_package/" + deviceName + fileSuffix); err != nil {
		log.Debug("找不到指定设备的升级文件tar.gz，使用默认文件")
		b, _ = os.ReadFile("./update_package/" + "device00000" + fileSuffix)
	}
	//写入到响应流中
	rsp.Write(b)
}

// 创建tar.gz文件
func createTarGz(sourceDir string, targetFile string) error {
	// 创建目标文件
	target, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer target.Close()

	// 创建gzip压缩器
	gzipWriter := gzip.NewWriter(target)
	defer gzipWriter.Close()

	// 创建tar文件写入器
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// 遍历源目录中的所有文件
	err = filepath.Walk(sourceDir, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录
		if fileInfo.IsDir() {
			return nil
		}

		// 打开文件
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// 创建tar文件头
		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return err
		}

		// 修改头部名称为文件名，去掉文件名中的路径信息
		header.Name = fileInfo.Name()

		// 设置修改时间和权限为固定值
		header.ModTime = time.Unix(0, 0)
		header.Mode = 0644

		// 写入头部信息
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 写入文件内容
		if _, err := io.Copy(tarWriter, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
