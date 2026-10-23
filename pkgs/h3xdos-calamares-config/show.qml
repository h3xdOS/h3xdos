/* h3xdOS installation slideshow (Calamares slideshow API 2).
 * Based on the Calamares example slideshow, GPL-3.0-or-later.
 */
import QtQuick 2.0;
import calamares.slideshow 1.0;

Presentation
{
    id: presentation

    readonly property color ink: "#1b1d22"
    readonly property color text: "#dfe3e8"
    readonly property color dust: "#7d8590"
    readonly property color signal: "#3281ea"
    readonly property int slideSeconds: 12

    Rectangle {
        anchors.fill: parent
        color: presentation.ink
        z: -1
    }

    Timer {
        interval: presentation.slideSeconds * 1000
        running: presentation.activatedInCalamares
        repeat: true
        onTriggered: presentation.goToNextSlide()
    }

    component Page: Slide {
        id: page
        property string heading
        property string body

        Column {
            anchors.centerIn: parent
            width: parent.width * 0.7
            spacing: 18

            Rectangle {
                width: 48
                height: 4
                color: presentation.signal
            }
            Text {
                text: page.heading
                color: presentation.text
                font.family: "JetBrainsMono Nerd Font"
                font.pixelSize: 26
                width: parent.width
                wrapMode: Text.WordWrap
            }
            Text {
                text: page.body
                color: presentation.dust
                font.pixelSize: 16
                width: parent.width
                wrapMode: Text.WordWrap
                lineHeight: 1.3
            }
        }
    }

    Page {
        heading: qsTr("Installing h3xdOS")
        body: qsTr("The live system is being copied to your disk. Depending on the disk this takes a few minutes; the computer can stay connected to the network.")
    }
    Page {
        heading: qsTr("The desktop")
        body: qsTr("Xfce with one panel at the top: the menu opens with the Super key, followed by workspaces, open windows, keyboard layout, network, sound and the clock.")
    }
    Page {
        heading: qsTr("Arch Linux packages")
        body: qsTr("pacman installs software from the Arch Linux repositories: sudo pacman -S name. sudo pacman -Syu updates the whole system, including the h3xdOS packages.")
    }
    Page {
        heading: qsTr("Ubuntu and Kali")
        body: qsTr("The menu entries Ubuntu terminal and Kali terminal open distrobox containers with apt and sudo. The first start downloads the container image.")
    }
    Page {
        heading: qsTr("After the restart")
        body: qsTr("Remove the installation medium when asked, then log in with the user created in this installer.")
    }

    function onActivate() {
        presentation.currentSlide = 0;
    }

    function onLeave() {
    }
}
