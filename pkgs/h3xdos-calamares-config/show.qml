/* H3XD installation slideshow (Calamares slideshow API 2), styled after sheet 2f.
 * Based on the Calamares example slideshow, GPL-3.0-or-later.
 */
import QtQuick 2.0;
import calamares.slideshow 1.0;

Presentation
{
    id: presentation

    readonly property color ground: "#181c25"
    readonly property color text: "#e6edf3"
    readonly property color body: "#c6cbd5"
    readonly property color accent: "#1793d1"
    readonly property int slideSeconds: 12

    Rectangle {
        anchors.fill: parent
        color: presentation.ground
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
            spacing: 16

            Rectangle {
                width: 24
                height: 4
                radius: 2
                color: presentation.accent
            }
            Text {
                text: page.heading
                color: presentation.text
                font.family: "IBM Plex Sans"
                font.weight: Font.DemiBold
                font.pixelSize: 26
                width: parent.width
                wrapMode: Text.WordWrap
            }
            Text {
                text: page.body
                color: presentation.body
                font.family: "IBM Plex Sans"
                font.pixelSize: 16
                width: parent.width
                wrapMode: Text.WordWrap
                lineHeight: 1.4
            }
        }
    }

    Page {
        heading: qsTr("Installing H3XD")
        body: qsTr("The live system is being copied to your disk. Depending on the disk this takes a few minutes; the computer can stay connected to the network.")
    }
    Page {
        heading: qsTr("The desktop")
        body: qsTr("Xfce with one bar at the top: the H3XD mark opens the menu, Super opens the search launcher, the bar shows workspaces, the active window, the clock, network, volume and battery.")
    }
    Page {
        heading: qsTr("Arch Linux packages")
        body: qsTr("pacman installs software from the Arch Linux repositories: sudo pacman -S name. System Update in the menu updates everything, including the H3XD packages from GitHub.")
    }
    Page {
        heading: qsTr("apt for Ubuntu and Kali")
        body: qsTr("sudo apt install name runs apt in an Ubuntu container and kali-apt does the same with Kali. The first start downloads the container image; installed apps appear in the menu.")
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
