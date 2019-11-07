package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const (
	mixerAPIBase = "https://mixer.com/api/v1"
	vlcPath      = "/Applications/VLC.app/Contents/MacOS/VLC"
)

var client = &http.Client{}
var token = flag.String("token", "", "Name of channel")

type channel struct {
	Token  string `json:"token"`
	ID     int    `json:"id"`
	Online bool   `json:"online"`
	Name   string `json:"name"`
	Type   struct {
		Name string `json:"name"`
	} `json:"type"`
}

func getChannel(token string) (channel, error) {
	var c channel

	resp, err := client.Get(fmt.Sprintf("%s/channels/%s", mixerAPIBase, token))
	if err != nil {
		return c, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&c)
	if err != nil {
		return c, err
	}

	return c, nil
}

func openVLC(done <-chan struct{}, manifestPath, title string) error {
	defer func() {
		fmt.Println("\rClosing VLC")
	}()

	cmd := exec.Command(vlcPath, fmt.Sprintf("--input-title-format=%s", title), manifestPath)

	go func() {
		<-done

		cmd.Process.Signal(syscall.SIGINT)
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func main() {
	defer func() {
		fmt.Println("\rGood Bye.")
	}()

	flag.Parse()

	if len(*token) == 0 {
		fmt.Println("Missing channel name - `mixer-vlc -token <user>`")
		return
	}

	c, err := getChannel(*token)
	if err != nil {
		fmt.Println("Error getting channel info", err)
		return
	}

	if !c.Online {
		fmt.Printf("%s is not online\n", c.Token)
		return
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs

		done <- struct{}{}
	}()

	err = openVLC(done, fmt.Sprintf("%s/channels/%v/manifest.m3u8", mixerAPIBase, c.ID), fmt.Sprintf("%s | %s | %s", c.Token, c.Name, c.Type.Name))
	if err != nil {
		fmt.Println("Error opening VLC", err)
		return
	}
}
