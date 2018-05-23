package nextcloud

import(
	"net/http"
	"io"
	"log"
	"os"
	"regexp"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"net/url"
)

// TODO Implement json.Marshaler
type Client struct{
	Username   string
	Password   string
	ServerUrl  string
	SyncFolder string
	httpClient *http.Client
}

func NewFromJson(jsonConfigPath string)  (*Client, error) {
	var client Client
	jsonConfig, err := ioutil.ReadFile(jsonConfigPath)
	if err != nil{
		return nil, err
	}
	json.Unmarshal(jsonConfig, &client)

	client.ServerUrl = client.ServerUrl + "remote.php/dav/files/" + client.Username + "/"
	client.httpClient = &http.Client{}
	return &client, nil
}


func NewFromParameters(username, password, serverUrl, syncFolder string) *Client{
	return &Client{username, password, serverUrl, syncFolder, &http.Client{}}
}


func (ncClient *Client) MkDirOnServer(dirPath string){
	response, _ := ncClient.execRequest("MKCOL", ncClient.GetRelativePath(dirPath, false), nil)
	response.Body.Close()
}

func (ncClient *Client) MkDirLocally(serverPath string){
	fullPath := ncClient.SyncFolder + ncClient.GetRelativePath(serverPath, true)
	os.MkdirAll(fullPath, os.ModePerm)
}

func (ncClient *Client) CopyToServer(localPath string) error{
	file, err := prepareFile(localPath)
	if err != nil{
		return err
	}
	pathOnServer := ncClient.GetRelativePath(localPath, false)
	response, err := ncClient.execRequest("PUT", pathOnServer, file)

	response.Body.Close()
	return err
}

func (ncClient *Client) CopyFromServer(serverPath string) (int, error){
	response, err := ncClient.execRequest("GET", serverPath, nil)
	if err != nil{
		// If we have a response, it may be a server error
		if response != nil {
			return response.StatusCode, err
		}
		return -1, err
	}
	defer response.Body.Close()
	return response.StatusCode, ncClient.downloadFile(response, serverPath)
}

func (ncClient *Client) Rm(localPath string){
	response, _ := ncClient.execRequest("DELETE", ncClient.GetRelativePath(localPath, false), nil)
	response.Body.Close()
}

/* Show contents on the remote server. Pass nil as the second argument if no extra data is necessary, otherwise
 * pass an io.Reader, eg:\n
  reader := strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
		<d:propfind xmlns:d="DAV:">
			<d:prop xmlns:oc="http://owncloud.org/ns">
				<d:getlastmodified/>
				<d:getcontentlength/>
				<d:getcontenttype/>
				<oc:fileid/>
				<oc:permissions/>
				<d:resourcetype/>
				<d:getetag/>
			</d:prop>
		</d:propfind>`)

  ncClient.Ls("/", reader) */
func (ncClient *Client) Ls(remotePath string, additionalInfo io.Reader) ([]byte, error) {
	response, err := ncClient.execRequest("PROPFIND", remotePath, additionalInfo)
	if err != nil{
		return nil, err
	}
	return ioutil.ReadAll(response.Body)
}

func (ncClient *Client) testConnection() bool{
	return false
}

func (ncClient *Client) HttpClient() *http.Client{
	return ncClient.httpClient
}

func (ncClient *Client) execRequest(method, path string, body io.Reader) (*http.Response, error){
	fullPath := ncClient.ServerUrl + path
	request, err := http.NewRequest(method, fullPath, body)
	if err != nil{
		log.Fatal(err)
		return nil, err
	} else {
		request.SetBasicAuth(ncClient.Username, ncClient.Password)
		response, err := ncClient.httpClient.Do(request)
		if err != nil{
			log.Fatal(err)
			return nil, err
		} else {
			return response, nil
		}
	}
}

// Given a full path to a file, it returns the path relative to the SyncFolder. If you need to get the relative path
// from a server path pass true to the second parameter, otherwise pass false eg:\n
// GetRelativePath("remote.php/dav/files/user/folder/file.txt", true) -> folder/file.txt
// GetRelativePath("/home/user/syncFolder/file.txt", false) -> file.txt
func (ncClient *Client) GetRelativePath(path string, fromServer bool) string{
	var folderPath string
	if fromServer {
		folderPath = fmt.Sprintf("remote.php/dav/files/%s/", ncClient.Username)
	} else {
		folderPath = ncClient.SyncFolder
	}

	pattern := regexp.MustCompile("(" + folderPath  + ")(.*)")
	relativePath := pattern.FindStringSubmatch(path)
	if len(relativePath) > 0{
		escapedPath, _ := url.PathUnescape(relativePath[2])
		return escapedPath
	}
	return ""
}

func (ncClient *Client) downloadFile(response *http.Response, serverPath string) error{
	fullPath := ncClient.SyncFolder + ncClient.GetRelativePath(serverPath, true)
	file, err := os.Create(fullPath)
	if err != nil{
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil{
		return err
	}
	return nil
}

func prepareFile(filePath string) (*os.File, error){
	file, err := os.Open(filePath)
	if err != nil{
		return nil, err
	}
	return file, nil
}

