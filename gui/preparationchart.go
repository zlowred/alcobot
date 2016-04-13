package gui

import (
	"fmt"

	"strings"
	"time"

	"math"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
	"github.com/zlowred/alcobot/series"
)

type PreparationChart struct {
	screen *RootScreen

	w    *ui.QWidget
	conf *config.Configuration

	targetTemp  *series.Series
	currentTemp *series.Series
	pid         *series.Series
	power       *series.Series
}

func NewPreparationChart(screen *RootScreen) *PreparationChart {
	c := &PreparationChart{screen: screen}
	c.w = ui.NewWidgetFromDriver(screen.FindChild("preparationChart"))

	c.w.InstallEventFilter(c)

	go c.loop()

	return c
}

func (c *PreparationChart) loop() {
	configCh := hub.JoinConfigGroup(c.screen.hub.Configuration)
	dpCh := hub.JoinDataPointGroup(c.screen.hub.DataPoints)

	for {
		select {
		case <-c.screen.hub.Quit:
			return
		case x := <-configCh:
			c.conf = x
			ui.Async(func() {
				c.w.Update()
			})
		case x := <-dpCh:
			if c.targetTemp == nil || c.currentTemp == nil || c.pid == nil || c.power == nil {
				continue
			}
			c.targetTemp.Push(x.TargetTemp)
			c.currentTemp.Push(x.CurrentTemp)
			c.pid.Push(x.PID)
			c.power.Push(x.Power)
			ui.Async(func() {
				c.w.Update()
			})
		}
	}
}

func (c *PreparationChart) OnPaintEvent(e *ui.QPaintEvent) bool {
	c.w.PaintEvent(e)

	if c.conf == nil {
		return true
	}

	if c.targetTemp == nil {
		c.targetTemp = series.NewSeries(int(c.w.Width() - sx))
	}
	if c.currentTemp == nil {
		c.currentTemp = series.NewSeries(int(c.w.Width() - sx))
	}
	if c.pid == nil {
		c.pid = series.NewSeries(int(c.w.Width() - sx))
	}
	if c.power == nil {
		c.power = series.NewSeries(int(c.w.Width() - sx))
	}

	painter := ui.NewPainterWithPaintDevice(c.w)
	defer painter.Delete()

	pen := ui.NewPen()
	pen.SetWidth(0)
	pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_white))
	painter.SetPenWithPen(pen)

	painter.DrawLineWithX1Y1X2Y2(sx, c.w.Height()-sy, c.w.Width(), c.w.Height()-sy)
	painter.DrawLineWithX1Y1X2Y2(sx, 0, sx, c.w.Height()-sy)
	painter.DrawLineWithX1Y1X2Y2(sx-25, 0, sx-25, c.w.Height()-sy)

	var i int32
	for i = 0; i < steps*2; i++ {
		px := sx + (c.w.Width()-sx)/(steps*2)*i
		painter.DrawLineWithX1Y1X2Y2(px, c.w.Height()-sy, px, c.w.Height()-sy+5+(5*((i+1)%2)))
		py := c.w.Height() - sy - (c.w.Height()-sy)/(steps*2)*i
		painter.DrawLineWithX1Y1X2Y2(sx, py, sx-5-(5*((i+1)%2)), py)
		painter.DrawLineWithX1Y1X2Y2(sx-25, py, sx-25-5-(5*((i+1)%2)), py)
	}

	pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_darkGray))
	pen.SetStyle(ui.Qt_DotLine)
	painter.SetPenWithPen(pen)

	for i = 1; i < steps; i++ {
		px := sx + (c.w.Width()-sx)/steps*i
		painter.DrawLineWithX1Y1X2Y2(px, c.w.Height()-sy, px, 0)
		py := c.w.Height() - sy - (c.w.Height()-sy)/steps*i
		painter.DrawLineWithX1Y1X2Y2(sx, py, c.w.Width(), py)
	}

	painter.SetRenderHint(ui.QPainter_Antialiasing)
	pen.SetStyle(ui.Qt_SolidLine)
	painter.SetPenWithPen(pen)

	min1, max1 := -100., 100.
	scale1 := float64(c.w.Height()-sy) / (max1 - min1)
	shift1 := (max1 + min1) / 2

	data := c.pid.Get()
	draw(data, shift1, scale1, ui.Qt_yellow, painter, float64(c.w.Width()), float64(c.w.Height()))
	data = c.power.Get()
	draw(data, shift1, scale1, ui.Qt_red, painter, float64(c.w.Width()), float64(c.w.Height()))

	data = c.currentTemp.Get()
	for i, _ := range data {
		if math.IsNaN(data[i]) {
			continue
		}
		if c.conf.TemperatureScale == config.F {
			data[i] = conv.DsToF(int16(data[i]))
		} else {
			data[i] = conv.DsToC(int16(data[i]))
		}
	}
	data2 := c.targetTemp.Get()
	for i, _ := range data2 {
		if math.IsNaN(data2[i]) {
			continue
		}
		if c.conf.TemperatureScale == config.F {
			data2[i] = conv.CtoF(data2[i])
		}
	}
	shift3, scale3, min2, max2 := analyze(append(data, data2...), float64(c.w.Height()-sy), 5)
	draw(data, shift3, scale3, ui.Qt_cyan, painter, float64(c.w.Width()), float64(c.w.Height()))

	draw(data2, shift3, scale3, ui.Qt_blue, painter, float64(c.w.Width()), float64(c.w.Height()))

	totalTime := (time.Now().Sub(c.conf.BrewingStartTime))
	for i = 0; i < steps; i++ {
		pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_gray))
		painter.SetPenWithPen(pen)
		painter.Save()
		font := painter.Font()
		defer font.Delete()
		font.SetPointSize(int32(10))
		painter.SetFont(font)
		painter.TranslateFWithDxDy(float64(sx+i*(c.w.Width()-sx)/steps), float64(c.w.Height()-sy+10))
		painter.Rotate(15)
		t := totalTime / time.Duration(steps) * time.Duration(i)
		t -= t % time.Minute
		s := strings.Replace(t.String(), "0s", "", -1)
		if i != 0 && s == "0" {
			s = ""
		}
		painter.DrawTextWithXYText(0, 0, s)
		painter.Restore()

		pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_green))
		painter.SetPenWithPen(pen)
		painter.Save()
		font = painter.Font()
		defer font.Delete()
		font.SetPointSize(int32(10))
		painter.SetFont(font)
		painter.TranslateFWithDxDy(float64(sx-5), float64(c.w.Height()-sy-(c.w.Height()-sy)/steps*i)-5)
		painter.Rotate(-105)
		painter.DrawTextWithXYText(0, 0, fmt.Sprintf("%.0f%%", min1+(max1-min1)/float64(steps)*float64(i)))
		painter.Restore()

		pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_yellow))
		painter.SetPenWithPen(pen)
		painter.Save()
		font = painter.Font()
		defer font.Delete()
		font.SetPointSize(int32(10))
		painter.SetFont(font)
		painter.TranslateFWithDxDy(float64(sx-5-25), float64(c.w.Height()-sy-(c.w.Height()-sy)/steps*i)-5)
		painter.Rotate(-105)
		if c.conf != nil && c.conf.TemperatureScale == config.F {
			painter.DrawTextWithXYText(0, 0, fmt.Sprintf("%.0fºF", min2+(max2-min2)/float64(steps)*float64(i)))
		} else {
			painter.DrawTextWithXYText(0, 0, fmt.Sprintf("%.0fºC", min2+(max2-min2)/float64(steps)*float64(i)))
		}
		painter.Restore()
	}

	pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_white))
	painter.SetPenWithPen(pen)

	return true
}
