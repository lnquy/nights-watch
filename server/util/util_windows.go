package util

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modadvapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procRegEnumValueW = modadvapi32.NewProc("RegEnumValueW")
)

func GetHome() string {
	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return home
}

func LineBreak() string {
	return "\r\n"
}

// Taken from https://github.com/bugst/go-serial
func nativeGetPorts() ([]string, error) {
	subKey, err := syscall.UTF16PtrFromString("HARDWARE\\DEVICEMAP\\SERIALCOMM\\")
	if err != nil {
		return nil, err
	}

	var h syscall.Handle
	if err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subKey, 0, syscall.KEY_READ, &h); err != nil {
		return nil, err
	}
	defer syscall.RegCloseKey(h)

	var valuesCount uint32
	if err = syscall.RegQueryInfoKey(h, nil, nil, nil, nil, nil, nil, &valuesCount, nil, nil, nil, nil); err != nil {
		return nil, err
	}

	list := make([]string, valuesCount)
	for i := range list {
		var data [1024]uint16
		dataSize := uint32(len(data))
		var name [1024]uint16
		nameSize := uint32(len(name))
		if err = regEnumValue(h, uint32(i), &name[0], &nameSize, nil, nil, &data[0], &dataSize); err != nil {
			return nil, err
		}
		list[i] = syscall.UTF16ToString(data[:])
	}
	return list, nil
}

func regEnumValue(key syscall.Handle, index uint32, name *uint16, nameLen *uint32, reserved *uint32, class *uint16, value *uint16, valueLen *uint32) (regerrno error) {
	r0, _, _ := syscall.Syscall9(procRegEnumValueW.Addr(), 8, uintptr(key), uintptr(index), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(nameLen)), uintptr(unsafe.Pointer(reserved)), uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(value)), uintptr(unsafe.Pointer(valueLen)), 0)
	if r0 != 0 {
		regerrno = syscall.Errno(r0)
	}
	return
}
