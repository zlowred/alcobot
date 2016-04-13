package service

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/zlowred/goqt/ui"
)

type Screenshoter struct {
	widget *ui.QWidget
}

func NewScreenshoter(widget *ui.QWidget) *Screenshoter {
	s := &Screenshoter{widget}
	http.HandleFunc("/", s.takeScreenshot)
	http.ListenAndServe(":8080", nil)
	return s
}

func (s *Screenshoter) takeScreenshot(writer http.ResponseWriter, request *http.Request) {
	file, err := ioutil.TempFile("", "screenshot")
	if err != nil {
		log.Printf("Can't create temp file for screenshot: %v", err)
		return
	}
	if err != nil {
		log.Printf("Can't create temp file for screenshot: %v", err)
		return
	}
	defer file.Close()
	wait := make(chan bool)
	ui.Async(func() {
		pixmap := ui.NewPixmapWithWidthHeight(s.widget.Width(), s.widget.Height())
		defer pixmap.Delete()
		s.widget.Render(pixmap)
		pixmap.SaveWithFilename(file.Name() + ".png")
		pixmap.SaveWithFilename(file.Name())
		close(wait)
	})
	<-wait
	imgFile, err := os.OpenFile(file.Name()+".png", os.O_RDONLY, 0)
	defer imgFile.Close()

	defer os.Remove(file.Name())
	defer os.Remove(imgFile.Name())
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := imgFile.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Can't write screenshot to http: %v", err)
		}
		if n == 0 {
			break
		}

		// write a chunk
		if _, err := writer.Write(buf[:n]); err != nil {
			log.Printf("Can't write screenshot to http: %v", err)
		}
	}
}
