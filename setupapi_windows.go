// +build windows

package setupapi

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Handle uintptr

const InvalidHandle = ^Handle(0)

//go:generate go run mksyscall_windows.go -output zsetupapi_windows.go setupapi_windows.go

//sys	setupDiClassGuidsFromNameEx(ClassName string, guid *Guid, size uint32, required_size *uint32, machineName string, reserved uint32) (err error) = setupapi.SetupDiClassGuidsFromNameExW
//sys	setupDiGetClassDevsEx(ClassGuid *Guid, Enumerator *string, hwndParent uintptr, Flags uint32, DeviceInfoSet uintptr, MachineName string, reserved uint32) (handle Handle, err error) = setupapi.SetupDiGetClassDevsExW
//sys	setupDiEnumDeviceInfo(DeviceInfoSet Handle, MemberIndex uint32, DeviceInfoData *spDeviceInformationData) (err error) = setupapi.SetupDiEnumDeviceInfo
//sys	setupDiGetDeviceInstanceId(DeviceInfoSet Handle, DeviceInfoData *spDeviceInformationData, DeviceInstanceId unsafe.Pointer, DeviceInstanceIdSize uint32, RequiredSize *uint32) (err error) = setupapi.SetupDiGetDeviceInstanceIdW

type Guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type SPDeviceInformationData struct {
	spDeviceInformationData
	devInfo DevInfo
}

type spDeviceInformationData struct {
	cbSize    uint32
	ClassGuid Guid
	DevInst   uint32
	reserved  uintptr
}

func (g Guid) String() string {
	return fmt.Sprintf("{%06X-%04X-%04X-%04X-%X}", g.Data1, g.Data2, g.Data3, g.Data4[0:2], g.Data4[2:])
}

type DIGCF uint32

const (
	Default         DIGCF = 0x00000001
	Present         DIGCF = 0x00000002
	AllClasses      DIGCF = 0x00000004
	Profile         DIGCF = 0x00000008
	DeviceInterface DIGCF = 0x00000010
	InterfaceDevice DIGCF = 0x00000010
)

type Error int

func GetLastError() error {
	return Error(int(windows.GetLastError().(syscall.Errno)))
}

func (e Error) Error() string {
	buf := make([]uint16, 300)
	_, err := windows.FormatMessage(syscall.FORMAT_MESSAGE_FROM_SYSTEM, 0, uint32(e), 0, buf, nil)
	return fmt.Sprintf("Error: %d %#v %#v", int(e), windows.UTF16ToString(buf), err)
}

// SetupDiClassGuidsFromNameEx retrieves the GUIDs associated with the specified class name. This resulting list contains the classes currently installed on a local or remote computer.
func SetupDiClassGuidsFromNameEx(className string, machineName string) ([]Guid, error) {
	requiredSize := uint32(0)
	err := setupDiClassGuidsFromNameEx(className, nil, 0, &requiredSize, machineName, 0)

	rets := make([]Guid, requiredSize, requiredSize)
	err = setupDiClassGuidsFromNameEx(className, &rets[0], 1, &requiredSize, machineName, 0)
	return rets, err
}

type DevInfo Handle

// SetupDiEnumDeviceInfo returns a SP_DEVINFO_DATA structure that specifies a device information element in a device information set.
func (di DevInfo) EnumDeviceInfo(memberIndex uint32) (*SPDeviceInformationData, error) {
	did := spDeviceInformationData{}

	did.cbSize = uint32(unsafe.Sizeof(did))

	err := setupDiEnumDeviceInfo(Handle(di), memberIndex, &did)
	return &SPDeviceInformationData{
		spDeviceInformationData: did,
		devInfo:                 di,
	}, err
}

// InstanceID retrieves the device instance ID that is associated with a device information element
func (did *SPDeviceInformationData) InstanceID() (string, error) {
	requiredSize := uint32(0)
	err := setupDiGetDeviceInstanceId(Handle(did.devInfo), &did.spDeviceInformationData, nil, 0, &requiredSize)

	buff := make([]uint16, requiredSize)
	err = setupDiGetDeviceInstanceId(Handle(did.devInfo), &did.spDeviceInformationData, unsafe.Pointer(&buff[0]), uint32(len(buff)), &requiredSize)
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(buff[:]), err
}

// SetupDiGetClassDevsEx returns a handle to a device information set that contains requested device information elements for a local or a remote computer.
func SetupDiGetClassDevsEx(ClassGuid Guid, Enumerator string, hwndParent uintptr, Flags DIGCF, DeviceInfoSet uintptr, MachineName string, reserved uint32) (DevInfo, error) {
	enumerator := &Enumerator

	if Enumerator == "" {
		enumerator = nil
	}

	hDevInfo, err := setupDiGetClassDevsEx(&ClassGuid, enumerator, hwndParent, uint32(Flags), DeviceInfoSet, MachineName, 0)
	return DevInfo(hDevInfo), err
}
