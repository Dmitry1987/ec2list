package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/ryanuber/columnize"
)

var allInstancesPtr *ec2.DescribeInstancesOutput

// Maybe later will split to several functions and files,
// for now it checks keywords for filtering, if supplied, shows needed servers,
// if not, lists all.
func main() {

	// Make sure credentials file exists (windows! use HOME for linux/mac version)
	credentials := os.Getenv("HOME") + "/.aws/credentials"
	if _, err := os.Stat(credentials); os.IsNotExist(err) {
		fmt.Printf("No credentials file found at: %s", credentials)
		os.Exit(1)
	}

	// 1 arg is script filename, so check if we got a text query?
	// if not, this keywords slice will be EMPTY so later we skip filtering.
	keywords := os.Args[1:]

	// This will be passed to columnize formatter, we need to pass array of strings,
	// by default delimeter of columns is "pipe" character (can override)
	outputLines := []string{"Name | IP", "-------------------- | -----------"}

	// used when we verify cache exist and not expired yet
	needRefresh := false
	// cache expiration time
	cacheSeconds := 3600

	// check if we have cache by getting file status (atime,mtime, etc')
	cacheModTime, err := os.Stat("cache.gob")
	// if no file, we get err
	if err != nil {
		needRefresh = true
		//DEBUG fmt.Println("Error get cache file, will create new.")
	} else {
		if (cacheModTime.ModTime().Unix() + int64(cacheSeconds)) < time.Now().Unix() {
			//DEBUG fmt.Println("Cache expired, refreshing...")
			needRefresh = true
		}
	}

	// we don't have cache yet, or is expired, run 'ec2 describe' query...
	if needRefresh {
		// Call DescribeInstances, this loads data into "allInstancesPtr" pointer
		// Create an EC2 service object
		ec2 := ec2.New(session.New(), &aws.Config{Region: aws.String("us-west-2")})
		allInstancesPtr, err = ec2.DescribeInstances(nil)
		// error can happen if we have no "describe ec2" permission...
		if err != nil {
			panic(err)
		}
		// if all ok, save all this data to cache
		saveCache("cache.gob", &allInstancesPtr)
		//DEBUG fmt.Println("saved cache!")
	} else {
		// load cache from file
		//DEBUG fmt.Println("loading file...")
		loadCache("cache.gob", &allInstancesPtr)
	}

	// allInstancesPtr now has data of all servers
	// the "Reservations" array has "Instances", but we have only 1 server always per 1 Reservation.
	// I added second loop anyway, instead directly using Reservations[0], to support "multiple instances per Reservation" in future.
	for _, oneReservation := range allInstancesPtr.Reservations {
		for _, oneInstance := range oneReservation.Instances {

			// skip any item without public IP, we don't need them in output
			if oneInstance.PublicIpAddress == nil {
				continue
			}

			for _, currentInstanceTag := range oneInstance.Tags {
				if *currentInstanceTag.Key == "Name" {
					// check if we have words for filtering
					if len(keywords) > 0 {
						// we'll count that ALL keywords exist in Name, and not only one of them
						matchedWordsNum := 0
						// iterate keywords array and check Name tag contains each word
						for _, word := range keywords {
							// using lowercase to lowercase comparison, no matter what we have in Name tag, we'll find it
							lowerCaseNameTag := strings.ToLower(*currentInstanceTag.Value)
							if strings.Contains(lowerCaseNameTag, strings.ToLower(word)) {
								matchedWordsNum++
							}
						}

						if matchedWordsNum == len(keywords) {
							// Later can optionally add  *oneInstance.InstanceId to output...
							outputLines = append(outputLines, *currentInstanceTag.Value+" | "+*oneInstance.PublicIpAddress)
						}
					} else {
						// no keyword specified - then we add any instance to list
						outputLines = append(outputLines, *currentInstanceTag.Value+" | "+*oneInstance.PublicIpAddress)
					}
				}
			}
		}
	}

	fmt.Println(columnize.SimpleFormat(outputLines))
	//fmt.Println("> Total servers: ", len(allInstancesPtr.Reservations))

}

// Save and load functions took from "http://www.robotamer.com/code/go/gotamer/gob.html"
func saveCache(path string, object interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

func loadCache(path string, object interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}
