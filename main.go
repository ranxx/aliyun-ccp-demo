package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-ccp/ccppath-sdk/go/client"
)

/*
var ossConfig = new(client.Config).
    SetDomainId("your domain id").
    SetProtocol("https").
    SetAccessKeyId(os.Getenv("ACCESS_KEY_ID")).
    SetAccessKeySecret(os.Getenv("ACCESS_KEY_SECRET"))

	// initial runtimeOptions
var runtime = &client.RuntimeOptions{}
// initial akClient
var ossClient, _ = client.NewClient(ossConfig)
*/

var ccpConfig = &client.Config{}
var ccpRuntime = &client.RuntimeOptions{}
var ccpClient *client.Client

func init() {
	ccpConfig.SetDomainId("hz417")
	ccpConfig.SetProtocol("https")
	ccpConfig.SetAccessKeyId(os.Getenv("ACCESS_KEY_ID"))
	ccpConfig.SetAccessKeySecret(os.Getenv("ACCESS_KEY_SECRET"))
	err := (error)(nil)
	ccpClient, err = client.NewClient(ccpConfig)
	fmt.Println(err, ccpClient)
}

func ccpCreateFile(driveID, name, Type, parentFile, contentType string) (*client.CreateFileModel, error) {
	createFileReq := new(client.CCPCreateFileRequest).SetDriveId(driveID).SetName(path.Base(name)).SetType(Type).
		SetParentFileId(parentFile).SetContentType(contentType)
	return ccpClient.CreateFile((&client.CCPCreateFileRequestModel{}).SetBody(createFileReq), ccpRuntime)
}

func ccpPutFile(filePath, uploadURL string) (*http.Response, error) {
	content, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer content.Close()
	req, err := http.NewRequest("PUT", uploadURL, content)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "")
	return http.DefaultClient.Do(req)
}

func ccpUploadFile(fileName, savadir string) {
	fmt.Println("up ...", fileName, " save", savadir)
	// 创建文件
	response, err := ccpCreateFile("1", fileName, "file", "root", "text/plain")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if response == nil || response.Body == nil || len(response.Body.PartInfoList) <= 0 {
		return
	}

	// 上传文件
	res, err := ccpPutFile(fileName, *response.Body.PartInfoList[0].UploadUrl)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer res.Body.Close()

	// complete file
	Etag := res.Header.Get("ETag")
	uploadPartInfo := new(client.UploadPartInfo).SetEtag(Etag).SetPartNumber(*response.Body.PartInfoList[0].PartNumber)
	completeFileReq := new(client.CCPCompleteFileRequest).SetDriveId("1").SetFileId(*response.Body.FileId).
		SetUploadId(*response.Body.UploadId).SetPartInfoList([]*client.UploadPartInfo{uploadPartInfo})
	_, err = ccpClient.CompleteFile((&client.CCPCompleteFileRequestModel{}).SetBody(completeFileReq), ccpRuntime)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

// life
func listFile() {
	listFileReq := new(client.CCPListFileRequest).
		SetDriveId("1").
		SetParentFileId("5e81d505b51860c2fe584cd0b026c97d358d2e26").
		SetLimit(10)
	listFileRes, err := ccpClient.ListFile((&client.CCPListFileRequestModel{}).SetBody(listFileReq), ccpRuntime)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(listFileRes)
}

// 有bug 如果文件名是 xx. 就不一定搜的到
func ccpSearchFile(fileName string) (*client.SearchFileModel, error) {
	// 取出名字和后缀
	cond := fmt.Sprintf(`name = '%s'`, fileName)
	if i := strings.LastIndex(fileName, "."); i != -1 {
		extName := fileName[i+1:]
		cond = fmt.Sprintf(`name = "%s" and file_extension in ["%s"]`, fileName[:i], extName)

	}
	searchFileReq := new(client.CCPSearchFileRequest).
		SetDriveId("1").
		SetLimit(10).
		SetQuery(cond)
	return ccpClient.SearchFile((&client.CCPSearchFileRequestModel{}).SetBody(searchFileReq), ccpRuntime)
}

func ccpDownloadURL(ccpFileID string) string {
	req := new(client.CCPGetDownloadUrlRequest).
		SetDriveId("1").
		SetFileId(ccpFileID)
	resp, err := ccpClient.GetDownloadUrl((&client.CCPGetDownloadUrlRequestModel{}).SetBody(req), ccpRuntime)
	if err != nil {
		return ""
	}
	if resp == nil || resp.Body == nil {
		return ""
	}
	return *resp.Body.Url
}

func downURLSavaLocalFileName(ccpFileURL, fileName string) {
	resp, err := http.Get(ccpFileURL)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// fmt.Println(string(data))
	save, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer save.Close()
	_, err = save.Write(data)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func ccpDownload(ccpFileName, saveDir string) {
	fmt.Println("down ...", ccpFileName, " save", saveDir)
	sfresp, err := ccpSearchFile(ccpFileName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// fmt.Println("ccpDownload", sfresp)
	if sfresp == nil || sfresp.Body == nil || len(sfresp.Body.Items) <= 0 {
		// 没有文件
		fmt.Printf("云盘没有%s文件\n", ccpFileName)
		return
	}

	ccpFileID := ""
	for _, v := range sfresp.Body.Items {
		if *(v.Name) == ccpFileName {
			ccpFileID = *sfresp.Body.Items[0].FileId
		}
	}

	if ccpFileID == "" {
		// 没有文件
		fmt.Printf("云盘没有%s文件\n", ccpFileName)
		return
	}

	// 获取下载地址
	url := ccpDownloadURL(ccpFileID)
	if url == "" {
		// 没有文件
		fmt.Printf("获取云盘%s文件下载地址有误\n", ccpFileName)
		return
	}

	if len(saveDir) > 0 {
		saveDir += "/"
	}

	// 下载
	saveDir = filepath.Dir(saveDir)
	if saveDir != "/" {
		saveDir = saveDir + "/" + ccpFileName
	}
	downURLSavaLocalFileName(url, saveDir)
}

func main() {
	// listDrive()
	ccpUploadFile("./main.go", "root")
	// listFile()
	// 等待5s 保证下载能生效
	time.Sleep(time.Second * 5)
	ccpDownload("main.go", "./test")
	// tips := "\n\t上传: up 文件路径\n\t下载: down 云端文件名 本地目录\n"
	// // do("https://ccp-hz417-hz-1585552672.oss-cn-hangzhou.aliyuncs.com/5e81dd42dbf712c7a4c1411e8e913dea746fdf44%2F5e81dd42f28ce1ae12484c659a258cc3dab3239e?Expires=1585580814\u0026OSSAccessKeyId=LTAIsE5mAn2F493Q\u0026Signature=OuUoO5Nmk%2BxB9g5AhHTyUXawJMw%3D")
	// for {
	// 	// 上传 up 本机文件名
	// 	// 下载 down 云端文件名
	// 	//
	// 	fmt.Println(tips)
	// 	fmt.Printf("→  ")
	// 	cmd := ""
	// 	fmt.Scanln(&cmd)
	// 	cmdFunc(cmd)
	// }
}

func cmdFunc(cmd string) {
	cmds := strings.Split(cmd, " ")
	if len(cmds) < 2 || (cmds[0] != "up" && cmds[0] != "down") {
		return
	}
	fmt.Println(cmds)
	switch cmds[0] {
	case "up":
		ccpUploadFile(cmds[1], "root")
	case "down":
		if len(cmds) < 3 {
			cmds = append(cmds, "./")
		}
		ccpDownload(cmds[1], cmds[2])
	default:
	}
}
