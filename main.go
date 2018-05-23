package main

import (
	"fmt"
    "os"
	"nextCloudClient/first_start"
	"nextCloudClient/inotify"
	"nextCloudClient/nextcloud"
)


func main() {
	USER := os.Getenv("USER")
	configPath := "/home/" + USER + "/.config/NextcloudClient/"
	_, err := os.Stat(configPath + "config.json")
	if err != nil{
		first_start.FirstStartSetup()
	} else {
		ncClient, err := nextcloud.NewFromJson("/home/" + USER + "/.config/NextcloudClient/config.json")
		if err != nil {
			fmt.Println("Something weird happened while I was trying to open your config file :(")
		}
		inotify.StartWatcherOnFolder(ncClient)
	}
}
