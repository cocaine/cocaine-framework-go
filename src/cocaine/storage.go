package cocaine

import (
//"codec"
//"fmt"
)

//Storage implementation
type Storage struct {
	*Service
}

const (
	STORAGE_READ = iota
	STORAGE_WRITE
	STORAGE_REMOVE
	STORAGE_FIND
)

//[0:read 1:write 2:remove 3:find]
func NewStorage(host string, port uint64) (*Storage, error) {
	return &Storage{NewService(host, port, "storage")}, nil
}

type StorageFindRes struct {
	Res []string
	Err error
}

// Try to make as template
func (storage *Storage) Find(namespace string, tags []string) chan StorageFindRes {
	Out := make(chan StorageFindRes)
	go func() {
		if res := <-storage.Call(STORAGE_FIND, namespace, tags); res.Err == nil {
			if res_as_strings, ok := res.Res.([]interface{}); ok {
				v := make([]string, 0, len(res_as_strings))
				for _, item := range res_as_strings {
					v = append(v, string(item.([]uint8)))
				}
				Out <- StorageFindRes{Res: v, Err: nil}
			} else {
				Out <- StorageFindRes{Res: nil, Err: &ServiceError{-100, "Invalid  result type"}}
			}
		} else {
			Out <- StorageFindRes{Res: nil, Err: res.Err}
		}
	}()
	return Out
}

type StorageReadRes struct {
	Res string
	Err error
}

func (storage *Storage) Read(namespace string, key string) chan StorageReadRes {
	Out := make(chan StorageReadRes)
	go func() {
		if res := <-storage.Call(STORAGE_READ, namespace, key); res.Err == nil {
			Out <- StorageReadRes{Res: string(res.Res.([]uint8)), Err: nil}
		} else {
			Out <- StorageReadRes{Res: "", Err: res.Err}
		}
	}()
	return Out
}

//type StorageWriteRes ServiceResult

func (storage *Storage) Write(namespace string, key string, data []byte, tags []string) chan ServiceResult {
	return storage.Call(STORAGE_WRITE, namespace, key, data, tags)
}

// type StorageRemoveRes struct {
// 	Res bool
// 	err error
// }

func (storage *Storage) Remove(namespace string, key string) chan ServiceResult {
	return storage.Call(STORAGE_REMOVE, namespace, key)
}
