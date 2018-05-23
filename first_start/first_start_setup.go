package first_start

import(
	"fmt"
	"bufio"
	"os"
	"golang.org/x/crypto/ssh/terminal"
	"strings"
	"encoding/json"
	"nextCloudClient/nextcloud"
	"container/list"
	"sync"
	"crypto/md5"
	"io/ioutil"
	"os/exec"
	"log"
)

var (
	USER = os.Getenv("USER")
	stack = nextcloud.NewFileStack()
)

func FirstStartSetup(){
	clearScreen()
	fmt.Println("\t\t##################################################")
	fmt.Println("\t\t\t\t\t NC")
	fmt.Println("\t\t##################################################")
	fmt.Println("\nWelcome! Before starting I'd need some information to connect to your NextCloud server.\nYour info " +
				"will be stored in ~/.config/NextcloudClient/config.json.\n\nIf you ever need to change any configuration, " +
				"just edit that file or simply delete it to start this startup wizard once again!")

	fmt.Println("\nFirst things first, I need your username and password to connect to the server.\nFret not though," +
				   "the config file won't be sent anywhere (except to your server, of course). HTTPS should be mandatory.")

	reader := bufio.NewReader(os.Stdin)
	username, password := getUsernameAndPassword(reader)

	fmt.Println("\nGreat, now I need your server url and the path of the local folder you want Nextcloud to be synced to.")
	serverUrl, syncFolder := getServerUrlAndSyncFolder(reader)

	writeToJson(username, password, serverUrl, syncFolder)
	fmt.Println("\nPerfect, everything's ready now. I'll start syncing your local folder with your server, this may" +
			    "take a while! Please be patient :D")

	SyncFolderWithServer(username, password, serverUrl, syncFolder)
	fmt.Println("\nGreat, now you should find all of your nextcloud files in your synced folder.\n" +
		"You can now launch this program as a daemon/systemd service. It'll keep your files in sync with your server,\n" +
		"bye! :D")
}


// See https://stackoverflow.com/questions/2137357/getpasswd-functionality-in-go
// for reference
func getUsernameAndPassword(reader *bufio.Reader) (username, password string){
	fmt.Println("username:\t")
	username, _ = reader.ReadString('\n')

	fmt.Println("password:\t")
	bytePassword, _ := terminal.ReadPassword(0)
	password = string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}


func getServerUrlAndSyncFolder(reader *bufio.Reader) (serverUrl, syncFolder string){
	fmt.Println("Enter your server url (eg. https://mydomain.com):\t")
	serverUrl, _ = reader.ReadString('\n')
	serverUrl = strings.TrimSpace(serverUrl)
	if !strings.HasSuffix(serverUrl, "/") {
		serverUrl = serverUrl + "/"
	}

	fmt.Println("\nOk, now enter the path of the folder to sync with NC (eg. /home/user/nextCloudFolder).\nIf it doesn't exist, I'll create it" +
		" for you\nPlease notice that if the folder contains any file/subfolder with the same name of anything downloaded" +
		" from the server, it will be overwritten.")
	syncFolder, _ = reader.ReadString('\n')

	syncFolder = strings.TrimSpace(syncFolder)
	if !strings.HasSuffix(syncFolder, "/") {
		syncFolder = syncFolder + "/"
	}
	os.MkdirAll(syncFolder, os.ModePerm)

	return
}


func writeToJson(username, password, serverUrl, syncFolder string) {
	jsonMap := map[string]string{
		"username": username,
		"password": password,
		"serverUrl": serverUrl,
		"syncFolder": syncFolder}

	path := fmt.Sprintf("/home/%s/.config/NextcloudClient/", USER)
	os.MkdirAll(path, os.ModePerm)
	file, err := os.Create(path + "config.json")
	if err != nil{
		panic(err)
	}
	defer file.Close()

	jsonBytes, _ := json.Marshal(jsonMap)
	file.Write(jsonBytes)
}


func SyncFolderWithServer(username, password, serverUrl, syncFolder string){
	client := nextcloud.NewFromParameters(username, password, serverUrl, syncFolder)

	dirChan := make(chan *list.List, 3)
	errChan := make(chan error)
	go getResponseFromServer("/remote.php/dav/files/" + username, dirChan, errChan, client)

	fmt.Println("Collecting info...")
	toCollect := 1
	for n := 0; n < toCollect; n++ {
		select {
		case dirList := <-dirChan:
			toCollect += dirList.Len()
			go func(){
				for d := dirList.Front(); d != nil; d = d.Next() {
					go getResponseFromServer(d.Value.(string), dirChan, errChan, client)
				}
			}()
		case err := <-errChan:
			fmt.Println(err)
			return
		}
	}
	close(dirChan)
	close(errChan)

	downloadFromStack(stack, client)
}

func getResponseFromServer(serverPath string, dirChan chan *list.List, errChan chan error, client *nextcloud.Client) {
	response, err := downloadInfo(serverPath, client)
	if err != nil {
		errChan <- err
	}

	fileList := list.New()
	for _, file := range response.Files {
		if !file.IsDir() {
			stack.Push(file.Href)
		} else {
			client.MkDirLocally(file.Href)
			fileList.PushFront(file.Href)
		}
	}
	dirChan <- fileList
}

func downloadInfo(path string, client *nextcloud.Client) (*nextcloud.Response, error) {
	response, err := client.Ls(path, nil)
	if err != nil {
		return nil, err
	}
	r, err := nextcloud.NewResponseFromXML(response)
	if err != nil{
		return r, err
	}
	r.RemoveFirstFile()
	return r, nil
}

func downloadFromStack(stack *nextcloud.FileStack, client *nextcloud.Client) {
	group := new(sync.WaitGroup)
	pathChan := make(chan string, 3)
	defer close(pathChan)

	group.Add(stack.Len())
	fmt.Printf("added %d\n", stack.Len())
	for el, err := stack.Pop(); err == nil; el, err = stack.Pop(){
		pathChan <- el
		go func() {
			downloadFile(<-pathChan, group, client)
		}()
	}
	group.Wait()
}

func downloadFile(path string, group *sync.WaitGroup, client *nextcloud.Client) {
	defer group.Done()
	fmt.Println("Downloading file: " + path)

	code, err := client.CopyFromServer(path)
	if err != nil {
		// Even if we got errors, we just keep downloading the next files.
		printDownloadErrorInfo(err, code, path, client.SyncFolder + client.GetRelativePath(path, true))
	} else {
		fmt.Println(path + " is done")
	}
}

func printDownloadErrorInfo(err error, code int, serverPath, localPath string) {
	if code >= 500{
		fmt.Printf("An error occurred on your nextcloud server, while downloading file %s\nHTTP status: %d", serverPath, code)
	} else {
		md5sum, err := getMD5Sum(localPath)
		errMsg := "Error occurred while downloading file " + serverPath + "\n(local path: " + localPath + ")\n"
		if err != nil {
			 errMsg += "\nA checksum wasn't possible, a manual download is recommended."
		} else {
			errMsg += fmt.Sprintf("\nSometimes it's still possible that the file has been downloaded correctly. Here's the checksum: %x\n\n", md5sum)
		}

		log.Println(errMsg)
	}
}

func getMD5Sum(localPath string) ([16]byte, error) {
	f, err := ioutil.ReadFile(localPath)
	return md5.Sum(f), err
}

// Works only on linux
func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}