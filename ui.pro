CONFIG      += plugin debug_and_release
TARGET      = $$qtLibraryTarget(mainscreenplugin)
TEMPLATE    = lib

HEADERS     =
SOURCES     =
RESOURCES   = alcobot.qrc
LIBS        += -L. -lmainscreen

greaterThan(QT_MAJOR_VERSION, 4) {
    QT += designer
} else {
    CONFIG += designer
}

target.path = $$[QT_INSTALL_PLUGINS]/designer
INSTALLS    += target

FORMS += \
    screens/root.ui


