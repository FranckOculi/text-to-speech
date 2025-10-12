// Package main is to interract with user.
// This package communicates with back server in order to process text to speech.

// VO

package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
)

var currentCmd *exec.Cmd
var mStop *systray.MenuItem
var uiActions = make(chan func())

func main() {
	go uiWorker()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Text To Speech")
	systray.SetTooltip("Text To Speech")

	data, err := os.ReadFile("/home/chouchou/Images/wave.svg")
	if err != nil {
		log.Println(err)
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
				text, err := getSelectedText()
				if err != nil {
					log.Printf("Error : %v", err)
					continue
				}
				if currentCmd != nil {
					stopSpeaking(false)
				}
				speak(text)
			case <-mStop.ClickedCh:
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
}

func getSelectedText() (string, error) {
	cmd := exec.Command("xclip", "-o", "-selection", "primary")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	log.Printf("--> %s\n", output)
	return string(output), nil
}

func speak(text string) {
	showStop()
	currentCmd = exec.Command("espeak-ng", "-v", "fr", text)
	if err := currentCmd.Start(); err != nil {
		log.Printf("Error TTS : %v\n", err)
		return
	}

	cmd := currentCmd
	go func() {
		cmd.Wait()
		if cmd == currentCmd {
			hideStop()
		}
	}()
}

func stopSpeaking(hide bool) {
	if currentCmd != nil && currentCmd.Process != nil {
		currentCmd.Process.Kill()
		currentCmd = nil
	}

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

// package main

// import (
// 	"log"
// 	"os"
// 	"os/exec"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/BurntSushi/xgb"
// 	"github.com/BurntSushi/xgb/xproto"
// 	"github.com/getlantern/systray"
// )

// var (
// 	conn     *xgb.Conn
// 	win      xproto.Window = 0
// 	lastText string
// 	quitChan = make(chan os.Signal, 1)
// )

// func main() {
// 	var err error
// 	conn, err = xgb.NewConn()
// 	if err != nil {
// 		log.Fatal("Erreur X11: ", err)
// 	}
// 	defer cleanup()

// 	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

// 	go func() {
// 		systray.Run(onReady, onExit)
// 	}()

// 	detectTextSelection()
// }

// func cleanup() {
// 	if conn != nil && win != 0 {
// 		xproto.DestroyWindow(conn, win)
// 	}
// 	if conn != nil {
// 		conn.Close()
// 	}
// }

// func onReady() {
// 	// 	// Titre de l'application (optionnel)
// 	systray.SetTitle("Lecteur de texte")
// 	// 	// Titre au survol de l'icône
// 	systray.SetTooltip("Lecteur de texte")
// 	systray.SetIcon(getIcon())
// }

// func onExit() {
// 	log.Println("Fermeture de l'application")
// 	quitChan <- syscall.SIGTERM
// }

// func getIcon() []byte {
// 	data, err := os.ReadFile("/home/chouchou/Images/wave.svg")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	return data
// }

// func detectTextSelection() {
// 	screen := xproto.Setup(conn).DefaultScreen(conn)

// 	for {
// 		select {
// 		case <-quitChan:
// 			return
// 		default:
// 			texte, err := getSelectedText()
// 			if err != nil {
// 				time.Sleep(200 * time.Millisecond)
// 				continue
// 			}

// 			if texte != "" && texte != lastText {
// 				lastText = texte
// 				if win == 0 {
// 					createFloatingWindow(screen)
// 				}
// 				updateWindowPosition(screen)
// 			} else if texte == "" && win != 0 {
// 				xproto.DestroyWindow(conn, win)
// 				win = 0
// 			}
// 			time.Sleep(100 * time.Millisecond)
// 		}
// 	}
// }

// func createFloatingWindow(screen *xproto.ScreenInfo) {
// 	var err error
// 	win, err = xproto.NewWindowId(conn)
// 	if err != nil {
// 		log.Println("Erreur création fenêtre: ", err)
// 		return
// 	}

// 	// Solution simplifiée : utiliser seulement CwBackPixel et CwEventMask
// 	xproto.CreateWindow(conn, screen.RootDepth, win, screen.Root,
// 		int16(0), int16(0), uint16(64), uint16(64), 0,
// 		xproto.WindowClassInputOutput, screen.RootVisual,
// 		xproto.CwBackPixel|xproto.CwEventMask,
// 		[]uint32{
// 			0xff0000, // Couleur de fond rouge
// 			xproto.EventMaskButtonPress,
// 		})

// 	// Définir override_redirect après la création
// 	xproto.ChangeWindowAttributes(conn, win, xproto.CwOverrideRedirect, []uint32{1})

// 	xproto.MapWindow(conn, win)
// }

// func updateWindowPosition(screen *xproto.ScreenInfo) {
// 	if win == 0 {
// 		return
// 	}

// 	_, err := xproto.GetGeometry(conn, xproto.Drawable(win)).Reply()
// 	if err != nil {
// 		log.Println("Fenêtre invalide, recréation...")
// 		xproto.DestroyWindow(conn, win)
// 		win = 0
// 		createFloatingWindow(screen)
// 		return
// 	}

// 	reply, err := xproto.QueryPointer(conn, screen.Root).Reply()
// 	if err != nil {
// 		log.Println("Erreur position curseur: ", err)
// 		return
// 	}

// 	xproto.ConfigureWindow(conn, win, xproto.ConfigWindowX|xproto.ConfigWindowY,
// 		[]uint32{uint32(reply.RootX + 20), uint32(reply.RootY + 20)})

// 	// Gestion des événements simplifiée
// 	ev, err := conn.PollForEvent()
// 	if err == nil {
// 		if e, ok := ev.(xproto.ButtonPressEvent); ok && e.Event == win && e.Detail == 1 {
// 			speak(lastText)
// 		}
// 	}
// }
// func getSelectedText() (string, error) {
// 	cmd := exec.Command("xclip", "-o", "-selection", "primary")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(output), nil
// }

// func speak(text string) {
// 	cmd := exec.Command("espeak-ng", "-v", "fr", text)
// 	err := cmd.Run()
// 	if err != nil {
// 		log.Println("Erreur TTS:", err)
// 	}
// }
