package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"front/utils"

	"github.com/getlantern/systray"
)

var (
	cancelFunc   context.CancelFunc
	cancelFuncMu sync.Mutex
	mStop        *systray.MenuItem
	currentCmd   *exec.Cmd
	currentCmdMu sync.Mutex
)

func main() {
	utils.LoadEnv()
	utils.InitLogger()
	log.Println("Start app")
	systray.Run(onReady, onExit)
}

func onReady() {
	mRead, mStop, mQuit := initSystray()

	go func() {
		for {
			select {
			case <-mRead.ClickedCh:
				log.Println("Read menu clicked")
				stopCurrentReading()

				ctx, cancel := context.WithCancel(context.Background())

				cancelFuncMu.Lock()
				cancelFunc = cancel
				cancelFuncMu.Unlock()

				// go readMP3File(ctx)
				go read(ctx)
				// go getTest(ctx)
			case <-mStop.ClickedCh:
				log.Println("Stop menu clicked")
				stopCurrentReading()
				mStop.Hide()
			case <-mQuit.ClickedCh:
				log.Println("Quit menu clicked")
				systray.Quit()
				return
			}
		}
	}()
}

func initSystray() (*systray.MenuItem, *systray.MenuItem, *systray.MenuItem) {
	systray.SetTitle("Text To Speech")
	systray.SetTooltip("Text To Speech")
	data, err := os.ReadFile("/home/chouchou/Pictures/wave.svg")
	if err != nil {
		log.Println("Error when reading app icon : ", err)
	} else {
		systray.SetIcon(data)
	}

	mRead := systray.AddMenuItem("Read", "Read selected text")
	mStop = systray.AddMenuItem("Stop", "Stop reading text")
	mStop.Hide()
	mQuit := systray.AddMenuItem("Quit", "Quit application")

	return mRead, mStop, mQuit
}

func onExit() {
	log.Println("Exiting application...")
	stopCurrentReading()
	log.Println("Cleanup done")
	time.Sleep(100 * time.Millisecond)
}

// func readMP3File(ctx context.Context) {
// 	cmd := exec.Command("mpg123", "output.mp3")

// 	if err := cmd.Run(); err != nil {
// 		log.Printf("Playback error: %v", err)
// 	}
// }

func getTest(ctx context.Context) {
	log.Println("Start Test...")

	mStop.Show()
	defer mStop.Hide()

	selectedText := utils.GetSelectedText()
	err := utils.VerifyText(selectedText)
	if err != nil {
		log.Println(err)
		return
	}

	requestBody := utils.RequestBody{Text: selectedText}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error JSON convert : %v\n", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:8080/test", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Request error : ", err)
		return
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Reading cancelled")
			return
		}

		log.Println("Request error : ", err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Println(res.Status)
		return
	}

	if res.StatusCode >= 400 {
		log.Println("Error response : ", res.Status, res.Body)
	}

	text, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
	}

	log.Println("Response : ", string(text))

	cancelFuncMu.Lock()
	cancelFunc = nil
	cancelFuncMu.Unlock()

	log.Println("Test finished")
}

func read(ctx context.Context) {
	log.Println("Start reading...")

	mStop.Show()
	defer mStop.Hide()

	select {
	case <-ctx.Done():
		log.Println("Reading cancelled")
		return
	default:
		text := utils.GetSelectedText()
		if text == "" {
			return
		}

		log.Printf("Selected text : %v \n", text)

		err := utils.VerifyText(text)
		if err != nil {
			log.Println(err)
			return
		}

		data, err := utils.GetSpeech(ctx, text)
		if err != nil {
			return
		}

		err = utils.SaveContent(data)
		if err != nil {
			return
		}

		cmd := exec.Command("mpg123", "output.mp3")
		// cmd := exec.CommandContext(ctx, "aplay", "output.wav")
		currentCmdMu.Lock()
		currentCmd = cmd
		currentCmdMu.Unlock()

		if err := cmd.Run(); err != nil {
			if ctx.Err() == context.Canceled {
				log.Println("Playback canceled")
			} else {
				log.Printf("Playback error: %v", err)
			}
		}

		cancelFuncMu.Lock()
		cancelFunc = nil
		cancelFuncMu.Unlock()

		log.Println("Reading finished")
	}
}

func stopCurrentReading() {
	currentCmdMu.Lock()
	if currentCmd != nil && currentCmd.Process != nil {
		currentCmd.Process.Signal(syscall.SIGTERM)
		currentCmd.Wait()
		currentCmd = nil
	}
	currentCmdMu.Unlock()

	cancelFuncMu.Lock()
	if cancelFunc != nil {
		log.Println("Stopping current reading")
		cancelFunc()
		cancelFunc = nil
	}
	cancelFuncMu.Unlock()
}
