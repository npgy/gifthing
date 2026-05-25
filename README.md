# Gifthing

## Format SD card

on macOS
`diskutil partitionDisk disk4 1 mbr FAT32 main R`

## Download Alpine

Download the tarball for raspberry pi aarch64
https://www.alpinelinux.org/downloads/

Extract it and copy it to the sd card

## Unattended setup

Copy `headless.apkovl.tar.gz` from https://github.com/macmpi/alpine-linux-headless-bootstrap
into the root of the SD card

This comes with a private key built in, you'll need to use it to SSH into the initial setup environment.
It's located at `tmp/.ALHB/ssh_host_rsa_key` inside the tarball

Copy all files in the `unattended_files` folder to the root of the SD card.  
Update `wpa_supplicant.conf` with your wifi SSID and password.

## Setup ALpine

Run `setup-alpine`

Enter `gifthing` for system name.  
Most other things will be the default value.  
Generate new SSH keypair.
`ssh-keygen -t ed25519`.
Paste the public key when the installer prompt asks for it.
Install in `sys` mode

Update apk packages  
`apk update && apk upgrade`

Copy `usercfg.txt` to the /boot directory

## Add some packages

You may need to enable the community repo.
Uncomment that line in `/etc/apk/repositories`

Now lets add some stuff

`apk add curl mpv ffmpeg jq`

If we have a gif file, first convert it to mp4
`ffmpeg -i punch.gif punch.mp4`

Use this to fit the video into the visible dimensions (the mat covers part of the displays width)
`./fitmat.sh punch.mp4 punch2.mp4`

do this for portrait/vertical gifs (verify the rotation once full product is built)
`ffmpeg -i stroop.mp4 -vf "transpose=1" stroop2.mp4`

Then have mpv play it
`mpv --loop=inf punch2.mp4`

Ideally though, the gifthing service will autoboot and play the file at `/root/main.mp4`. Once we build the web server, we will execute ffmpeg commands that generate the main.mp4 file

## Make booting cleaner

To add `gifthing` to the init system  
Copy it to `/etc/init.d/gifthing`

Add it to the default runlevel  
`rc-update add gifthing boot`  
Clear openrc cache (why is this needed lol)  
`rc-update -u`

Add `console=serial0 vt.global_cursor_default=0` to the end of `/boot/cmdline.txt`

Comment out all ttys in /etc/inittab

## Set the Wifi

On another computer, plug in the SD card,
then edit `/etc/wpa_supplicant/wpa_supplicant.conf`
with your wifi ssid and password

# Viewing images on the framebuffer

Install fbi and a monospace font  
`apk add fbida-fbi ttf-dejavu`

And here's how to display an image  
`fbi -a --noverbose img.webp`

## Local domain name resolution

Install avahi using these guides
https://wiki.alpinelinux.org/wiki/MDNS  
https://github.com/LouisBrunner/avahi2dns
