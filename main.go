//go:generate bash -c "if [ ! -f ./tools/go-bindata ]; then echo Building go-bindata tool...; fi"
//go:generate bash -c "if [ ! -f ./tools/go-bindata ]; then go build -o tools/go-bindata github.com/jteeuwen/go-bindata/go-bindata; fi"
//go:generate echo Compiling binary resources...
//go:generate ./tools/go-bindata -o hub/queries.go sql/...
//go:generate perl -pi -e s,main,hub,g hub/queries.go
//go:generate echo Compiling UI resources...
//go:generate $GOPATH/src/github.com/zlowred/goqt/bin/goqt_rcc -go main -o alcobot_qrc.go alcobot.qrc
//go:generate perl -pi -e s,visualfc,zlowred,g alcobot_qrc.go
//go:generate echo Done
package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/zlowred/goqt/ui"

	"github.com/zlowred/alcobot/backlight"
	"github.com/zlowred/alcobot/flightrecorder"
	"github.com/zlowred/alcobot/gui"
	"github.com/zlowred/alcobot/hal"
	"github.com/zlowred/alcobot/heatpump"
	"github.com/zlowred/alcobot/hub"
	"github.com/zlowred/alcobot/pid"
	"github.com/zlowred/alcobot/service"
)

func main() {
	if h, err := hub.New(); err != nil {
		panic(err)
	} else {
		if pid, err := pid.New(100, 0.35, 0.3, -255, 255, h); err != nil {
			panic(err)
		} else {
			hl := hal.New(h)
			go func() {
				sigchan := make(chan os.Signal, 10)
				signal.Notify(sigchan)
				<-sigchan
				for i := 0; i < 10; i++ {
					hl.ResetI2C()
				}
				h.Quit <- true
			}()
			defer func() {
				for i := 0; i < 10; i++ {
					hl.ResetI2C()
				}
			}()
			heatpump.New(h)
			backlight.New(h)
			flightrecorder.New(h)

			ui.Run(func() {
				w, err := gui.NewRootScreen(h)
				if err != nil {
					panic(err)
				}

				w.Show()

				go func() {
					<-h.Quit
					w.Close()
				}()
				go func() {
					time.Sleep(time.Second)
					pid.Enable()
					service.NewScreenshoter(w.QWidget)
				}()
			})
		}

	}
}
