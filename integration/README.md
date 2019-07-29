# Integration Tests

This tests core use cases for webboot such as:

- retrieving and kexec'ing a Linux kernel,
- uinit (user init), and
- running unit tests requiring root privileges.

## Usage

Run the tests with:

    go test

     or 

    UROOT_QEMU=qemu-system-x86_64 UROOT_KERNEL=$HOME/linux/arch/x86/boot/bzImage go test

When the QEMU arch is not amd64, set the `UROOT_TESTARCH` variable. For
example:
    UROOT_TESTARCH=arm go test

Currently, only amd64 and arm are supported.

- Steps to get the kernel (bzImage)
- Step 1: clone the open source linux directory:
  - git clone https://github.com/torvalds/linux.git
  - cd linux 

- Step 2: checkout the latest Linux version 
  - git checkout v4.17

- Step 3: Download a config file from u-root and replace the current .config in linux with u-root .config file
  - wget https://raw.githubusercontent.com/u-root/u-root/master/.circleci/images/test-image-amd64/config_linux4.17_x86_64.txt


- Step 4: build the kernel 
  - Make -j12 

- Step 5: Drop a linux kernel inside of the integration folder and run the RUNLOCAL binary
					OR

- Step 5: if you do not want to drop the kernel, run the command "UROOT_QEMU=qemu-system-x86_64 UROOT_KERNEL=$HOME/linux/arch/x86/boot/bzImage go test" 



## Requirements
  - Make -j12 

- QEMU
  - Path and arguments must be set with `UROOT_QEMU`.
  - Example: `export UROOT_QEMU="/usr/bin/qemu-system-x86_64"`
- Linux kernel
  - Path and arguments must be set with `UROOT_KERNEL`.
  - Example: `export UROOT_KERNEL="$HOME/linux/arch/x86/boot/bzImage"`
