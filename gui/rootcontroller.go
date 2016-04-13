package gui

import (
	"math"
	"time"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/hub"
)

const animationLength = 300
const easingType = ui.QEasingCurve_OutBounce
const height = 470

type RootController struct {
	screen *RootScreen

	setupScreenHeight       float64
	preparationScreenHeight float64
	brewingScreenHeight     float64

	setupBtn       *ui.QPushButton
	preparationBtn *ui.QPushButton
	brewingBtn     *ui.QPushButton
	quitBtn        *ui.QPushButton
	screenLayout   *ui.QVBoxLayout

	setupScreen       *ui.QWidget
	preparationScreen *ui.QWidget
	brewingScreen     *ui.QWidget

	conf *config.Configuration
}

func animateHeight(w ui.QObjectInterface, old float64, new float64, parent ui.QObjectInterface) {
	a := ui.NewPropertyAnimationWithObjectPropertynameParent(w, []byte("maximumHeight"), parent)
	a.SetDuration(animationLength)
	c := ui.NewEasingCurveWithType(easingType)
	a.SetEasingCurve(c)
	s := ui.NewVariantWithFloat64(old)
	a.SetStartValue(s)
	e := ui.NewVariantWithFloat64(new)
	a.SetEndValue(e)
	a.Start()

	go func() {
		<-time.NewTimer(time.Millisecond * 2 * animationLength).C
		ui.Async(func() {
			a.Delete()
			c.Delete()
			s.Delete()
			e.Delete()
		})
	}()
}

func (ctl *RootController) loop() {
	screenCh := hub.JoinScreenGroup(ctl.screen.hub.ScreenChange)
	configCh := hub.JoinConfigGroup(ctl.screen.hub.Configuration)
	for {
		select {
		case <-ctl.screen.hub.Quit:
			return
		case x := <-screenCh:
			switch x {
			case config.SETUP_SCREEN:
				ui.Async(func() {
					ctl.setSetupScreen()
				})
			case config.PREPARATION_SCREEN:
				ui.Async(func() {
					ctl.setPreparationScreen()
				})
			case config.BREWING_SCREEN:
				ui.Async(func() {
					ctl.setBrewingScreen()
				})
			}
		case x := <-configCh:
			ctl.conf = x
			ui.Async(func() {
				ctl.preparationBtn.SetEnabled(len(ctl.conf.FermenterSensor) > 0 && math.Abs(ctl.conf.NpaCalibration) > 0 && ctl.conf.Stage != config.BREWING)
				ctl.brewingBtn.SetEnabled(ctl.conf.Stage >= config.BREWING)
			})
		}
	}
}

func (ctl *RootController) setSetupScreen() {
	animateHeight(ctl.setupScreen, ctl.setupScreenHeight, height, ctl.screenLayout)
	animateHeight(ctl.preparationScreen, ctl.preparationScreenHeight, 0, ctl.screenLayout)
	animateHeight(ctl.brewingScreen, ctl.brewingScreenHeight, 0, ctl.screenLayout)

	ctl.setupBtn.SetChecked(true)
	ctl.preparationBtn.SetChecked(false)
	ctl.brewingBtn.SetChecked(false)

	ctl.setupScreenHeight, ctl.preparationScreenHeight, ctl.brewingScreenHeight = height, 0, 0
}

func (ctl *RootController) setPreparationScreen() {
	animateHeight(ctl.setupScreen, ctl.setupScreenHeight, 0, ctl.screenLayout)
	animateHeight(ctl.preparationScreen, ctl.preparationScreenHeight, height, ctl.screenLayout)
	animateHeight(ctl.brewingScreen, ctl.brewingScreenHeight, 0, ctl.screenLayout)

	ctl.setupBtn.SetChecked(false)
	ctl.preparationBtn.SetChecked(true)
	ctl.brewingBtn.SetChecked(false)

	ctl.setupScreenHeight, ctl.preparationScreenHeight, ctl.brewingScreenHeight = 0, height, 0
}

func (ctl *RootController) setBrewingScreen() {
	animateHeight(ctl.setupScreen, ctl.setupScreenHeight, 0, ctl.screenLayout)
	animateHeight(ctl.preparationScreen, ctl.preparationScreenHeight, 0, ctl.screenLayout)
	animateHeight(ctl.brewingScreen, ctl.brewingScreenHeight, height, ctl.screenLayout)

	ctl.setupBtn.SetChecked(false)
	ctl.preparationBtn.SetChecked(false)
	ctl.brewingBtn.SetChecked(true)

	ctl.setupScreenHeight, ctl.preparationScreenHeight, ctl.brewingScreenHeight = 0, 0, height
}

func NewRootController(screen *RootScreen) *RootController {
	ctl := &RootController{screen: screen}

	ctl.setupBtn = ui.NewPushButtonFromDriver(screen.FindChild("setupBtn"))
	ctl.preparationBtn = ui.NewPushButtonFromDriver(screen.FindChild("preparationBtn"))
	ctl.brewingBtn = ui.NewPushButtonFromDriver(screen.FindChild("brewingBtn"))
	ctl.quitBtn = ui.NewPushButtonFromDriver(screen.FindChild("quitBtn"))
	ctl.setupScreen = ui.NewWidgetFromDriver(screen.FindChild("setupScreen"))
	ctl.setupScreenHeight = float64(ctl.setupScreen.MaximumHeight())
	ctl.preparationScreen = ui.NewWidgetFromDriver(screen.FindChild("preparationScreen"))
	ctl.preparationScreenHeight = float64(ctl.preparationScreen.MaximumHeight())
	ctl.brewingScreen = ui.NewWidgetFromDriver(screen.FindChild("brewingScreen"))
	ctl.brewingScreenHeight = float64(ctl.brewingScreen.MaximumHeight())
	ctl.screenLayout = ui.NewVBoxLayoutFromDriver(screen.FindChild("screenLayout"))

	ctl.setupBtn.SetChecked(ctl.setupScreenHeight > 0)
	ctl.preparationBtn.SetChecked(ctl.preparationScreenHeight > 0)
	ctl.brewingBtn.SetChecked(ctl.brewingScreenHeight > 0)

	ctl.setupBtn.OnClicked(func() {
		ctl.setSetupScreen()
	})
	ctl.preparationBtn.OnClicked(func() {
		ctl.setPreparationScreen()
	})
	ctl.brewingBtn.OnClicked(func() {
		ctl.setBrewingScreen()
	})
	ctl.quitBtn.OnClicked(func() {
		close(screen.hub.Quit)
	})

	go ctl.loop()

	return ctl
}
