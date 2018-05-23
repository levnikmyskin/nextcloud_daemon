package inotify

import (
	"github.com/fsnotify/fsnotify"
	"regexp"
	"nextCloudClient/nextcloud"
	"log"
)

var ignoreList = [4]string{"part", "swp", "pyc", "class"}

/**
 * The events are treated this way:
 * If a CREATE event is triggered, the file is sent to the server
 * If a WRITE event is triggered, the file is sent to the server
 * If a RENAME event is triggered, the file is removed from the server
 * Same as above for REMOVE events
 * Files with several extensions are ignored (eg. .part, .swp etc.)
 */
func ManageEvent(event fsnotify.Event, ncClient *nextcloud.Client, watcher *fsnotify.Watcher){
	if fileIsRelevant(event){
		log.Println("event:", event)
		execActionOnServer(event, ncClient, watcher)
	}
}


// If the list will grow much larger, we might use
// an hashmap. At the moment it doesn't provide any consistent benefit
func fileIsRelevant(event fsnotify.Event) bool{
	if isDirOp(event.Op) {
		return true
	}
	re := regexp.MustCompile(`(/.*/.+)(?P<extension>\..+)`)
	matches := re.FindStringSubmatch(event.Name)

	if len(matches) > 0 {
		extension := matches[len(matches) - 1]
		for _, ignoreStr := range ignoreList {
			pattern := regexp.MustCompile(ignoreStr)
			if pattern.MatchString(extension) {
				return false
			}
		}
	}
	return true
}


func isDirOp(op fsnotify.Op) bool{
	return op&fsnotify.IsDir == fsnotify.IsDir
}


func execActionOnServer(event fsnotify.Event, client *nextcloud.Client, watcher *fsnotify.Watcher){
	if isDirOp(event.Op){
		execDirAction(event, client, watcher)
	} else {
		execFileAction(event, client)
	}
}


func execDirAction(event fsnotify.Event, client *nextcloud.Client, watcher *fsnotify.Watcher){
	op := event.Op
	if op&fsnotify.Create == fsnotify.Create{
		client.MkDirOnServer(event.Name)
		watcher.Add(event.Name)
	} else if op&fsnotify.Remove == fsnotify.Remove || op&fsnotify.Rename == fsnotify.Rename{
		client.Rm(event.Name)
	}
}


func execFileAction(event fsnotify.Event, client *nextcloud.Client){
	op := event.Op
	if op&fsnotify.Create == fsnotify.Create || op&fsnotify.Write == fsnotify.Write{
		client.CopyToServer(event.Name)
	} else if op&fsnotify.Remove == fsnotify.Remove || op&fsnotify.Rename == fsnotify.Rename{
		client.Rm(event.Name)
	}
}
