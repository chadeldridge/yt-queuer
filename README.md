# yt-queuer
yt-queuer is a lightweight service to queue YouTube videos for playback on remote in browser client.

## New Features
### Version 0.1.0
- Playback client setup. You will need to provide a name for the playback client the first time you open the playback in a browser.
- Playback clients now use a custom ID instead of a UUID. The new ID is generated from the playback client name and will be the same each time a device with that name is registered allowing you to easily recover playback clients or have multiple playback clients pulling from the same playlist.
- yt-queuer now supports multiple playlists. A new playlist is created for each uniquely named playback client. Playback clients with the same name will all play from the same playlist. You may 
- Power - Power button in the top right now supports HDMI CEC and Wake On LAN. Settings are stored per Playback Client.
  - Currently only supports one target per playback device.
  - HDMI CEC Controls - If the device ytqueuer is running on is connected via HDMI to a device that supports CEC, you can now get the power stutus, turn the power on, or put the connected device into standby.
  - Wake On LAN Support - You can now send a Wake On LAN packet from the controller page.

## Planned Features and TODOs
- Add controls for playback client (play, pause, seek, next). LOE - Low (figure out callback/socket)
- Create a browser addon. Add options in standard right-click browser menu to add a video to a playlist rather than having to go to the controll page to do it. LOE - High
- Move to React. If feel like much of the controller frontend would work better as a stateful React app. LOE Medium
- Investigate sqlc.
- Support running multiple yt-queuer instances as remote devices controlled by a primary instance. This would allow control of multiple TV endpoints from a single controller page.

## Installation
### Release
You can get the pre-compiled packages [here](https://github.com/chadeldridge/release).

### Build It Yourself
The Makefile build targets will build binary in ```bin/``` then copy the binary and any other needed files to ```pkg/``` where they will be tarred and added to ```repo/``` as ```repo/ytqueuer-[os]-[arch].tar.gz```.

Using the default Makefile build target:
```sh
make build
```

OR

Use a predefined build target for a tested architecture.
```sh
make build-arm64
```
Will build for arm64. You will need the appropriate gcc cross-compiler installed for this. In the case of build-arm64 being run from an Ubuntu amd64 device:
```sh
sudo apt install gcc-aarch64-linux-gnu
```

## Install
If you want to have HDMI CEC controls you need to make sure you have cec-ctl installed and have everything setup with a /dev/cec0 device or similar. If you are running the Raspberry Pi OS then this is probably already done. If not, I recommend checking out Arch Linux's [HDMI-CEC page](https://wiki.archlinux.org/title/HDMI-CEC).

I've provided an ```INSTALL.sh``` script which is designed for debian systems running systemd.
```
$ ./INSTALL.sh --help
must be run as root, use sudo
Usage: ./INSTALL.sh [-u user] [-h home]
  -u user: User to create. Default ytqueuer
  -h home: Full path of the desired home directory for ytqueuer. Default: /opt/ytqueuer
```
The installer will create the user, move all of the files into the home directory, add permissions to sudoers and add the user to the ```video``` group if it exists, and add the ytqueuer.service to systemd. Finally, if all went well, it will start the ytqueuer service through systemd.

The sudoer lines allow running the included ```wol``` binary as root. This is needed to allow the broadcast of Wake On LAN packets. If you don't plan to use Wake On LAN then I recommend removing these lines from sudoers.

The ```video``` group, if setup properly, allows ytqueuer access to run cec-ctl as root. If the group does not exist, the install script will ignore it and you will have to setup permission yourself unless you do not plan to use this feature.

## Run
The first time ytqueuer runs it will create a new 'certs' folder with a self generated key and cert if they do not already exist. ytqueuer must use ssl to allow clipboard access in the controller.
```sh
certs/certificate.crt
certs/privatekey.key
```
You can replace these certs with your own if you wish or back them up to restore later. If you wish to backup the database is will be locaded in the ytqueuer home directory under ```db/```.

## Access
From your preferred browser on the host you want to play videos on, go to:
```
https://localhost:8080
```

If your playback client is different from the device you are running ytqueuer on, replace 'localhost' with a reachable interface on the ytqueuer host.

You will be prompted to give the playback client a name. If you name multiple playback clients with the same name they will all play from the same playlist with the same name.

The first time you load the page you will need to accept the self-signed cert on the warning page (click on Advanced). You will also need to allow Auto Play in the Address Bar. Once Auto Play has been enabled, the playback page should automatically advance to the first video in the queue.

The browser should load a default video with 'under construction' tape across it if there is nothing in the queue.

## Control
From a browser go to:
```
https://<ytqueuer-host-ip>:8080/controller.html
```

![Controller Page](doc/ytqueuer_controller-page.png)

If you have already setup a playback device you can select it in the top bar to see the playlist.

Once a playlist is selected, any videos in the queue they will be displayed in the middle of the page. On the left you will see icons to:

![Add](doc/ytqueuer_controller-playlist_add.png) Add a video to the queue.

![Add Next](doc/ytqueuer_controller-playlist_add-to-top.png) Add a video to played next in queue.

![Clear Queue](doc/ytqueuer_controller-playlist_clear.png) Clear the queue. NO CONFIRMATION

When using Add and Add Next you need to have a YouTube video URL copied to your clipboard. When you click on the button you will be prompted to Paste which will take a few seconds to highlight. (this is a Firefox security feature to prevent accidental clicks) The Add and Add Next buttons will take the video URL form your clickboard, get the video id and send it to the API.

To the right of each video in the queue you will see a Remove From Queue button. There is also no confirmation on this button.

## Contributing
Contributions are welcome! Please fork the repository and submit a pull request.

## License
This project is licensed under the MIT License. See the `LICENSE` file for more details.

## Contact
For any questions or issues, please open an issue on the GitHub repository.