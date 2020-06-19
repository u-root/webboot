package main

import (
        "flag"
        "fmt"
        "io/ioutil"
        "log"
        "os"
        "path/filepath"

        "github.com/u-root/u-root/pkg/mount"
        "github.com/u-root/webboot/pkg/menus"
)

//kernelPath is for hardcode the path to kernel. should be replaced by u-root later
var kernelPath = make(map[string]string)

var (
        isoDir = flag.String("dir", ".", "set the iso directory path")
)

//ISO type contains information of a iso file, incluing its name, its filepath, the path to its kernel and if it should be choose by default.
type ISO struct {
        name      string
        path      string
        kernel    string
        isDefault bool
}

//IsDefault is for mark whether this iso is a default choise.
func (u ISO) IsDefault() bool {
        return u.isDefault
}

func (u ISO) String() string {
        return fmt.Sprintf("%s, %s, %v\n", u.name, u.path, u.isDefault)
}

//Label is the string this iso displays in the menu page.
func (u ISO) Label() string {
        return u.name
}

//getKernel is to find the kernel of a ISO. if the path of ISO is given, just check if the kernel is exist.
func getKernel(u *ISO) error {

        log.Println("try mount iso...")

        isoName := u.name

        diskFile, err := ioutil.TempDir("", "/mnt-iso")
        if err != nil {
                log.Fatal(err)
        }
        defer os.RemoveAll(diskFile)

        mountPath := filepath.Join(diskFile, isoName)
        if mp, err := mount.Mount(u.path, mountPath, "iso9660", "", mount.ReadOnly); err != nil {
                log.Printf("TryMount %s = %v, want nil", u.path, err)
        } else {
                log.Printf("mounted disk file is %s\n", mountPath)
                //if kernel is given, check its validation
                if u.kernel != "" {
                        walkfunc := func(path string, info os.FileInfo, err error) error {
                                if path == filepath.Join(mountPath, u.kernel) {
                                        log.Printf("Kernel at %s is found\n", u.kernel)
                                }
                                return nil
                        }

                        log.Println("\nfinding kernel...")
                        filepath.Walk(diskFile, walkfunc)
                        log.Printf("Kernel path is %s\n", u.kernel)
                }

                log.Println("\ntry unmount iso...")
                if err := mp.Unmount(0); err != nil {
                        log.Printf("Unmount(%q) = %v, want nil", mountPath, err)
                }
        }

        log.Println("Done")
        return nil
}

//isoMenu is to  build the iso menu page and return the iso path and kernel path of which user choose.
func isoMenu(isos []ISO) (string, string) {

        coord := 0

        var entries []menus.Entry
        for _, iso := range isos {
                entries = append(entries, iso)
        }

        index, err := menus.GetChoose(&coord, "Choose an iso you want to boot (hit enter to choose the default - 1 is the default option) >", entries)

        if err != nil || index < 0 || index >= len(isos) {
                if err != nil {
                        log.Println(err)
                }
                return "", ""
        }

        var chosenISO = isos[index]

        if err := getKernel(&chosenISO); err != nil {
                log.Println(err)
                return "", ""
        }

        return chosenISO.path, chosenISO.kernel
}

func main() {
        flag.Parse()
        
        //hardcode kernel path
        kernelPath["archlinux-2020.06.01-x86_64.iso"] = "arch/boot/x86_64/vmlinuz"
        kernelPath["TinyCore-11.1.iso"] = "boot/vmlinuz"

        var isos []ISO
        
        //find all iso inside the given directory
        walkfunc := func(path string, info os.FileInfo, err error) error {
                if info.IsDir() == false && filepath.Ext(path) == ".iso" {
                        var iso = ISO{info.Name(), path, kernelPath[info.Name()], true}
                        isos = append(isos, iso)
                }
                return nil
        }

        filepath.Walk(*isoDir, walkfunc)

        isoPath, isoKernel := isoMenu(isos)
        log.Printf("the choose iso is %s, the kernel path is %s\n", isoPath, isoKernel)
        return
}