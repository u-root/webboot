# Roadmap

Last updated June 23, 2021.

* **Directory server**: Instead of storing the list of distros in
  cmds/webboot/types.go, download the list as a json file over HTTP (possibly
  from github).
* **Improve UI error messaging**: Devise a better paradigm to display useful
  error messages to the user while still having highly technical logs
  available.
* **ISO builder**: Add a tool to automatically build the ISO image. Right now,
  you have to get comfortable with fdisk and dd in order to install webboot.
* **Get rid of C-based kexec**
* **CentOS 8**
* **ISO signing support**
