package azurerm

func azureRMUnlockMultiple(names *[]string) {
	for _, name := range *names {
		armMutexKV.Unlock(name)
	}
}
func azureRMLockMultiple(names *[]string) {
	for _, name := range *names {
		armMutexKV.Lock(name)
	}
}
