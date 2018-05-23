package inotify


import(
	"log"
	"github.com/fsnotify/fsnotify"
	"nextCloudClient/nextcloud"
)



func StartWatcherOnFolder(ncClient *nextcloud.Client){
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				go ManageEvent(event, ncClient, watcher)
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(ncClient.SyncFolder)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
