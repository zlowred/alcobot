package gui

import (
	"errors"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/hub"
)

type RootScreen struct {
	*ui.QWidget

	hub *hub.Hub

	rootController        *RootController
	brewingController     *BrewingController
	settingsController    *SettingsController
	preparationController *PreparationController
	brewingChart          *BrewingChart
	preparationChart      *PreparationChart
}

func NewRootScreen(hub *hub.Hub) (*RootScreen, error) {
	screen := &RootScreen{hub: hub}

	file := ui.NewFileWithName(":/screens/root.ui")
	defer file.Delete()

	if !file.Open(ui.QIODevice_ReadOnly) {
		return nil, errors.New("error loading ui resource")
	}

	loader := ui.NewUiLoader()
	defer loader.Delete()

	widget := loader.Load(file)
	if widget == nil {
		return nil, errors.New("error loading ui from resource")
	}

	screen.QWidget = widget

	screen.rootController = NewRootController(screen)
	screen.brewingController = NewBrewingController(screen)
	screen.settingsController = NewSettingsController(screen)
	screen.brewingChart = NewBrewingChart(screen)
	screen.preparationController = NewPreparationController(screen)
	screen.preparationChart = NewPreparationChart(screen)

	return screen, nil
}
