package configuration

import (
	"io/ioutil"
	"os"
)

const (
	filePermissions = 0600
	dirPermissions  = 0700
)

//go:generate counterfeiter . Persistor

type Persistor interface {
	Delete()
	Exists() bool
	Load(DataInterface) error
	Save(DataInterface) error
}

//go:generate counterfeiter . DataInterface

type DataInterface interface {
	JSONMarshalV3() ([]byte, error)
	JSONUnmarshalV3([]byte) error
}

type DiskPersistor struct {
	filePath string
}

func NewDiskPersistor(path string) DiskPersistor {
	return DiskPersistor{
		filePath: path,
	}
}

func (dp DiskPersistor) Exists() bool {
	_, err := os.Stat(dp.filePath)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

func (dp DiskPersistor) Delete() {
	_ = os.Remove(dp.filePath)
}

func (dp DiskPersistor) Load(data DataInterface) error {
	err := dp.read(data)
	if os.IsPermission(err) {
		return err
	}

	if err != nil {
		err = dp.write(data)
	}
	return err
}

func (dp DiskPersistor) Save(data DataInterface) error {
	return dp.write(data)
}

func (dp DiskPersistor) read(data DataInterface) error {
	err := dp.makeDirectory()
	if err != nil {
		return err
	}

	jsonBytes, err := ioutil.ReadFile(dp.filePath)
	if err != nil {
		return err
	}

	err = data.JSONUnmarshalV3(jsonBytes)
	return err
}

func (dp DiskPersistor) write(data DataInterface) error {
	bytes, err := data.JSONMarshalV3()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dp.filePath, bytes, filePermissions)
	return err
}
