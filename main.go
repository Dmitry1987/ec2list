package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"bytes"
	"strings"
	"strconv"
	"time"
	"os/user"
	"os/exec"
	"io/ioutil"
	"bufio"
	"golang.org/x/crypto/ssh"
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

	// Make sure credentials file exists 
	usr, _ := user.Current()
	credentials := usr.HomeDir + "/.aws/credentials"
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
	serverID := []string{ "none", "none" }

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
		// Create an EC2 service object. 
		// Here I hardcoded my needed region, but it can be configurable from external file like ~/.aws/config
		// Feel free to fork and make this tool production ready for your needs :) .
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

	// will be used as prefix number, for quick ssh to servers in list
	listOrder := 0

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
							listOrder++
							serverID = append(serverID, *oneInstance.PublicIpAddress)
							outputLines = append(outputLines, strconv.Itoa(listOrder) + ") " + *currentInstanceTag.Value+" | "+*oneInstance.PublicIpAddress)
						}
					} else {
						// no keyword specified - then we add any instance to list
						listOrder++
						serverID = append(serverID, *oneInstance.PublicIpAddress)
						outputLines = append(outputLines, strconv.Itoa(listOrder) + ") " + *currentInstanceTag.Value+" | "+*oneInstance.PublicIpAddress)
					}
				}
			}
		}
	}

	fmt.Println(columnize.SimpleFormat(outputLines))
	//fmt.Println("> Total servers: ", len(allInstancesPtr.Reservations))

	fmt.Print("Chose server to ssh: ")
    inputReader := bufio.NewReader(os.Stdin)
    serverNumber, err := inputReader.ReadString('\n')
    if err != nil {
        fmt.Println("An error occurred while reading user input")
    }
    // remove line break, we need only number
    //serverNumber = serverNumber[:len(serverNumber)-1]
    serverNumber = strings.Trim(serverNumber, "\r\n")
    
    num, err := strconv.Atoi(serverNumber)
 	if err != nil {
        fmt.Println("Probably not entered number")
    }
    // we need to address our server arrays with +2, because first 2 items were headers for columnize
    // and placeholders in serverID, so their length will be same, to avoid confusion when addressing only IP or full lines 
    fmt.Print("Connecting to  [ (" + outputLines[num + 1] + " ]")
    
	// pass the IP from serverID array that holds IPs by position order like numbered list we displayed
    // for now I decided to exec external ssh client whatever installed on user machine
	cmd := exec.Command("ssh","-o UserKnownHostsFile=null","-o StrictHostKeyChecking=no","-p 22", "user@"+serverID[num + 1] )
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

    // here is another option, to use golang ssh library, with all 'Session' features. It might help in further versions,
    // if i'll need to implement a loop over many servers with getting some info from them, or running commands.
	//sshLogin(serverID[num + 1]) 
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

func getKeyFile() (key ssh.Signer, err error){
    usr, _ := user.Current()
    file := usr.HomeDir + "/.ssh/id_rsa"
    //fmt.Println(file)
    buf, err := ioutil.ReadFile(file)
    if err != nil {
    	fmt.Println("error")
        return
    }
    key, err = ssh.ParsePrivateKey(buf)
    if err != nil {
    	fmt.Println("error")
        return
     }

    return
}

func sshLogin(ip string) {
	// call getKeyFile to detect ssh keys in home folder
	// if no keys found, panic will exit here
	key, err := getKeyFile()
	
	if err != nil {
     panic(err)
	} else {
		fmt.Println("DEBUG: Found SSH key")
	}

	config := &ssh.ClientConfig{
	    User: "user",
	    Auth: []ssh.AuthMethod{
	    ssh.PublicKeys(key),
	    },
	}

	client, err := ssh.Dial("tcp", ip + ":22", config)

	if err != nil {
    	panic("Failed to dial: "+ err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
	    panic("Failed to create session: " + err.Error())
	}
	defer session.Close()


	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("/usr/bin/whoami"); err != nil {
	    panic("Failed to run: " + err.Error())
	}
	fmt.Println("Result of ssh command")
	fmt.Println(b.String())
}