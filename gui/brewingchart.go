package gui

import (
	"fmt"
	"math"
	"time"

	"strings"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
	"github.com/zlowred/alcobot/series"
)

const sx int32 = 75
const sy int32 = 35
const steps int32 = 6

type BrewingChart struct {
	screen *RootScreen

	w             *ui.QWidget
	adsSeries     *series.Series
	dsSeries      *series.Series
	npaPresSeries *series.Series

	conf *config.Configuration
}

func NewBrewingChart(screen *RootScreen) *BrewingChart {
	c := &BrewingChart{screen: screen}
	c.w = ui.NewWidgetFromDriver(screen.FindChild("brewingChart"))

	c.w.InstallEventFilter(c)

	go c.loop()

	return c
}

func (c *BrewingChart) loop() {
	adsValueSensor := hub.JoinInt16Group(c.screen.hub.AdsValueSensor)
	dsTemp := hub.JoinInt16Group(c.screen.hub.DsTemperatureFiltered)
	npaPres := hub.JoinInt16Group(c.screen.hub.NpaPressureFiltered)
	configCh := hub.JoinConfigGroup(c.screen.hub.Configuration)

	for {
		select {
		case <-c.screen.hub.Quit:
			return
		case x := <-adsValueSensor:
			if c.adsSeries == nil {
				break
			}
			c.adsSeries.Push((float64(x) - 4500))
			ui.Async(func() {
				c.w.Update()
			})
		case x := <-dsTemp:
			if c.conf == nil {
				continue
			}
			if c.dsSeries == nil {
				break
			}
			c.dsSeries.Push(float64(x))
			ui.Async(func() {
				c.w.Update()
			})
		case x := <-npaPres:
			if c.conf == nil {
				continue
			}
			if c.npaPresSeries == nil {
				break
			}
			c.npaPresSeries.Push(conv.NpaToPa(x, c.conf.NpaZero, c.conf.NpaMinValue, c.conf.NpaMaxValue, c.conf.NpaMinPressure, c.conf.NpaMaxPressure))
			ui.Async(func() {
				c.w.Update()
			})
		case x := <-configCh:
			c.conf = x
		}
	}
}

func (c *BrewingChart) OnPaintEvent(e *ui.QPaintEvent) bool {
	c.w.PaintEvent(e)

	if c.adsSeries == nil {
		c.adsSeries = series.NewSeries(int(c.w.Width() - sx))
	}
	if c.dsSeries == nil {
		c.dsSeries = series.NewSeries(int(c.w.Width() - sx))
	}
	if c.npaPresSeries == nil {
		c.npaPresSeries = series.NewSeries(int(c.w.Width() - sx))
	}

	painter := ui.NewPainterWithPaintDevice(c.w)
	defer painter.Delete()

	//painter.FillRectWithXYWidthHeightGlobalcolor(0, 0, c.w.Width(), c.w.Height(), ui.Qt_black)
	pen := ui.NewPen()
	pen.SetWidth(0)
	pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_white))
	painter.SetPenWithPen(pen)

	painter.DrawLineWithX1Y1X2Y2(sx, c.w.Height()-sy, c.w.Width(), c.w.Height()-sy)
	painter.DrawLineWithX1Y1X2Y2(sx, 0, sx, c.w.Height()-sy)
	painter.DrawLineWithX1Y1X2Y2(sx-25, 0, sx-25, c.w.Height()-sy)
	painter.DrawLineWithX1Y1X2Y2(sx-50, 0, sx-50, c.w.Height()-sy)

	var i int32
	for i = 0; i < steps*2; i++ {
		px := sx + (c.w.Width()-sx)/(steps*2)*i
		painter.DrawLineWithX1Y1X2Y2(px, c.w.Height()-sy, px, c.w.Height()-sy+5+(5*((i+1)%2)))
		py := c.w.Height() - sy - (c.w.Height()-sy)/(steps*2)*i
		painter.DrawLineWithX1Y1X2Y2(sx, py, sx-5-(5*((i+1)%2)), py)
		painter.DrawLineWithX1Y1X2Y2(sx-25, py, sx-25-5-(5*((i+1)%2)), py)
		painter.DrawLineWithX1Y1X2Y2(sx-50, py, sx-50-5-(5*((i+1)%2)), py)
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

	data := c.adsSeries.Get()
	shift1, scale1, min1, max1 := analyze(data, float64(c.w.Height()-sy), 100)
	draw(data, shift1, scale1, ui.Qt_green, painter, float64(c.w.Width()), float64(c.w.Height()))

	data = c.dsSeries.Get()
	for i, _ := range data {
		if c.conf.TemperatureScale == config.F {
			data[i] = conv.DsToF(int16(data[i]))
		} else {
			data[i] = conv.DsToC(int16(data[i]))
		}
	}
	shift2, scale2, min2, max2 := analyze(data, float64(c.w.Height()-sy), 5)
	draw(data, shift2, scale2, ui.Qt_yellow, painter, float64(c.w.Width()), float64(c.w.Height()))

	data = c.npaPresSeries.Get()
	shift3, scale3, min3, max3 := analyze(data, float64(c.w.Height()-sy), 1)
	draw(data, shift3, scale3, ui.Qt_cyan, painter, float64(c.w.Width()), float64(c.w.Height()))

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
		painter.DrawTextWithXYText(0, 0, fmt.Sprintf("%.0f", min1+(max1-min1)/float64(steps)*float64(i)))
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

		pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_cyan))
		painter.SetPenWithPen(pen)
		painter.Save()
		font = painter.Font()
		defer font.Delete()
		font.SetPointSize(int32(10))
		painter.SetFont(font)
		painter.TranslateFWithDxDy(float64(sx-5-50), float64(c.w.Height()-sy-(c.w.Height()-sy)/steps*i)-5)
		painter.Rotate(-105)
		painter.DrawTextWithXYText(0, 0, fmt.Sprintf("%.0fPa", min3+(max3-min3)/float64(steps)*float64(i)))
		painter.Restore()
	}

	pen.SetColor(ui.NewColorWithGlobalcolor(ui.Qt_white))
	painter.SetPenWithPen(pen)

	return true
}

func draw(data []float64, shift float64, scale float64, color ui.Qt_GlobalColor, painter *ui.QPainter, width float64, height float64) {
	if len(data) > 0 {
		pen := ui.NewPen()
		defer pen.Delete()
		pen.SetColor(ui.NewColorWithGlobalcolor(color))
		pen.SetWidth(2)
		painter.SetPenWithPen(pen)
		path := ui.NewPainterPath()
		defer path.Delete()
		wasNan := true
		for x := len(data) - 1; x >= 0; x-- {
			if math.IsNaN(data[x]) {
				wasNan = true
				continue
			}
			if wasNan {
				path.MoveToFWithXY(width-float64(len(data)-1)+float64(x), height-float64(sy)-adj(data[x], shift, scale, height-float64(sy)))
				wasNan = false
			} else {
				path.LineToFWithXY(width-float64(len(data)-1)+float64(x), height-float64(sy)-adj(data[x], shift, scale, height-float64(sy)))
			}
		}
		painter.DrawPath(path)
	}
}

func adj(x float64, shift float64, scale float64, height float64) float64 {
	return (x-shift)*scale + height/2
}

func analyze(data []float64, height float64, rounding float64) (shift float64, scale float64, min float64, max float64) {
	min = math.MaxFloat64
	max = -math.MaxFloat64
	for _, d := range data {
		if math.IsNaN(d) {
			continue
		}
		min = math.Min(min, d)
		max = math.Max(max, d)
	}

	if max-min < 0.00001 {
		max += 0.00001
		min -= 0.00001
	}

	min = math.Floor(min/rounding) * rounding
	max = math.Ceil(max/rounding) * rounding

	scale = height / (max - min)
	shift = (max + min) / 2

	return shift, scale, min, max
}
