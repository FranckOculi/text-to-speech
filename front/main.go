// // Package main is to interract with user.
// // This package communicates with back server in order to process text to speech.

// V1

package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

var (
	interrupt    = make(chan os.Signal, 1)
	currentCmd   *exec.Cmd
	currentCmdMu sync.Mutex
	mStop        *systray.MenuItem
	uiActions    = make(chan func())
	cancelFunc   context.CancelFunc
	cancelFuncMu sync.Mutex
	wg           sync.WaitGroup
)

func main() {
	signal.Notify(interrupt, os.Interrupt)
	go uiWorker()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Text To Speech")
	systray.SetTooltip("Text To Speech")
	data, err := os.ReadFile("/home/chouchou/Images/wave.svg")
	if err != nil {
		log.Println("Error when reading app icon:", err)
	} else {
		systray.SetIcon(data)
	}
	mRead := systray.AddMenuItem("Read", "Read selected text")
	mStop = systray.AddMenuItem("Stop", "Stop reading text")
	hideStop()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		for {
			select {
			case <-mRead.ClickedCh:
				log.Println("Read clicked")
				text, err := getSelectedText()
				if err != nil {
					log.Printf("Error whn getting selected text: %v", err)
					continue
				}
				text = strings.TrimSpace(text)
				if text == "" {
					log.Println("no text selected")
					continue
				}

				stopSpeaking(false)
				time.Sleep(50 * time.Millisecond)
				speak(text)
			case <-mStop.ClickedCh:
				log.Println("Requesting stop")
				stopSpeaking(true)
			case <-mQuit.ClickedCh:
				log.Println("Requesting quit")
				systray.Quit()
				log.Println("Finished quitting")
				return
			}
		}
	}()
}

func onExit() {
	stopSpeaking(true)
	wg.Wait() // Wait to all goroutines
}

func getSelectedText() (string, error) {
	cmd := exec.Command("xclip", "-o", "-selection", "primary")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	text := string(output)
	log.Printf("Selected text: '%s'", text)
	return text, nil
}

func speak(text string) {
	ctx, cancel := context.WithCancel(context.Background())

	cancelFuncMu.Lock()
	cancelFunc = cancel
	cancelFuncMu.Unlock()

	showStop()
	log.Printf("Start reading text: '%s'", truncate(text, 50))

	cmd := exec.Command("aplay", "-f", "S16_LE", "-r", "16000")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Error when creating pipe for aplay:", err)
		hideStop()
		cancel()
		return
	}

	currentCmdMu.Lock()
	currentCmd = cmd
	currentCmdMu.Unlock()

	if err := cmd.Start(); err != nil {
		log.Println("Error when starting aplay:", err)
		hideStop()
		cancel()
		return
	}

	encodedText := url.QueryEscape(text)
	httpURL := "http://localhost:8080/tts?text=" + encodedText
	log.Printf("URL called: %s", truncate(httpURL, 80))

	wg.Add(1)

	go func(cmd *exec.Cmd, cancel context.CancelFunc) {
		defer wg.Done()
		defer hideStop()
		defer cancel()

		client := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", httpURL, nil)
		if err != nil {
			log.Printf("Error when creating http request: %v", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			if !isContextCancelError(err) {
				log.Printf("HTTP request error: %v", err)
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Server error: %s", resp.Status)
			return
		}

		bytesCopied, err := io.Copy(stdin, resp.Body)
		if err != nil {
			if !isBrokenPipeError(err) && !isContextCancelError(err) {
				log.Printf("Error streaming (%d octets copied): %v", bytesCopied, err)
			}
		} else {
			log.Printf("Streaming done (%d octets)", bytesCopied)
		}

		stdin.Close()

		currentCmdMu.Lock()
		currentCmdLocal := currentCmd
		currentCmdMu.Unlock()

		if currentCmdLocal != nil && currentCmdLocal.Process != nil {
			currentCmdLocal.Wait()
		}
	}(cmd, cancel)

}

func stopSpeaking(hide bool) {
	log.Println("Stop current speaking, hide=", hide)

	cancelFuncMu.Lock()
	if cancelFunc != nil {
		cancelFunc()
		cancelFunc = nil
	}
	cancelFuncMu.Unlock()

	currentCmdMu.Lock()
	if currentCmd != nil {
		if err := currentCmd.Process.Signal(syscall.SIGTERM); err != nil {
			currentCmd.Process.Kill()
		}
		currentCmd = nil
	}
	currentCmdMu.Unlock()

	if hide {
		hideStop()
	}
}

func uiWorker() {
	for fn := range uiActions {
		fn()
	}
}

func hideStop() {
	uiActions <- func() { mStop.Hide() }
}

func showStop() {
	uiActions <- func() { mStop.Show() }
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func isBrokenPipeError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "pipe broken") ||
		strings.Contains(err.Error(), "i/o timeout")
}

func isContextCancelError(err error) bool {
	if err == nil {
		return false
	}

	return err == context.Canceled ||
		strings.Contains(err.Error(), "context canceled") ||
		strings.Contains(err.Error(), "operation canceled")
}
