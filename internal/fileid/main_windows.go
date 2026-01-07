package fileid

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func queryFilenameById(path string) (uint64, error) {
	_path, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("convert path to UTF16 %s: %w", path, err)
	}
	handle, err := windows.CreateFile(_path,
		windows.GENERIC_READ,
		0,
		nil,
		windows.OPEN_EXISTING,
		0,
		0)

	if err != nil {
		return 0, fmt.Errorf("open file %s: %w", path, err)
	}
	defer windows.CloseHandle(handle)

	var data windows.ByHandleFileInformation

	if err = windows.GetFileInformationByHandle(handle, &data); err != nil {
		return 0, fmt.Errorf("get file information for %s: %w", path, err)
	}

	return (uint64(data.FileIndexHigh) << 32) | uint64(data.FileIndexLow), nil
}
