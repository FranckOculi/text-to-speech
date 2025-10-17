package main

import (
	"context"
	"log"
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

				go read(ctx)
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

func read(ctx context.Context) {
	log.Println("Start reading...")

	mStop.Show()
	defer mStop.Hide()

	if ctx.Err() != nil {
		log.Println("Reading cancelled")
		return
	}

	text := utils.GetSelectedText()
	if text == "" {
		return
	}

	log.Printf("Selected text : %v \n", text)

	data, err := utils.GetSpeech(ctx, text)
	if err != nil {
		return
	}

	err = utils.SaveContent(data)
	if err != nil {
		return
	}

	cmd := exec.CommandContext(ctx, "aplay", "output.wav")
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
