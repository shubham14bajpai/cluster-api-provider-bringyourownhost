// Code generated by counterfeiter. DO NOT EDIT.
package reconcilerfakes

import (
	"sync"
)

type FakeIK8sInstaller struct {
	InstallStub        func(string, string) error
	installMutex       sync.RWMutex
	installArgsForCall []struct {
		arg1 string
		arg2 string
	}
	installReturns struct {
		result1 error
	}
	installReturnsOnCall map[int]struct {
		result1 error
	}
	UninstallStub        func(string, string) error
	uninstallMutex       sync.RWMutex
	uninstallArgsForCall []struct {
		arg1 string
		arg2 string
	}
	uninstallReturns struct {
		result1 error
	}
	uninstallReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIK8sInstaller) Install(arg1 string, arg2 string) error {
	fake.installMutex.Lock()
	ret, specificReturn := fake.installReturnsOnCall[len(fake.installArgsForCall)]
	fake.installArgsForCall = append(fake.installArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	stub := fake.InstallStub
	fakeReturns := fake.installReturns
	fake.recordInvocation("Install", []interface{}{arg1, arg2})
	fake.installMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIK8sInstaller) InstallCallCount() int {
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	return len(fake.installArgsForCall)
}

func (fake *FakeIK8sInstaller) InstallCalls(stub func(string, string) error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = stub
}

func (fake *FakeIK8sInstaller) InstallArgsForCall(i int) (string, string) {
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	argsForCall := fake.installArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeIK8sInstaller) InstallReturns(result1 error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = nil
	fake.installReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIK8sInstaller) InstallReturnsOnCall(i int, result1 error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = nil
	if fake.installReturnsOnCall == nil {
		fake.installReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.installReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIK8sInstaller) Uninstall(arg1 string, arg2 string) error {
	fake.uninstallMutex.Lock()
	ret, specificReturn := fake.uninstallReturnsOnCall[len(fake.uninstallArgsForCall)]
	fake.uninstallArgsForCall = append(fake.uninstallArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	stub := fake.UninstallStub
	fakeReturns := fake.uninstallReturns
	fake.recordInvocation("Uninstall", []interface{}{arg1, arg2})
	fake.uninstallMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIK8sInstaller) UninstallCallCount() int {
	fake.uninstallMutex.RLock()
	defer fake.uninstallMutex.RUnlock()
	return len(fake.uninstallArgsForCall)
}

func (fake *FakeIK8sInstaller) UninstallCalls(stub func(string, string) error) {
	fake.uninstallMutex.Lock()
	defer fake.uninstallMutex.Unlock()
	fake.UninstallStub = stub
}

func (fake *FakeIK8sInstaller) UninstallArgsForCall(i int) (string, string) {
	fake.uninstallMutex.RLock()
	defer fake.uninstallMutex.RUnlock()
	argsForCall := fake.uninstallArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeIK8sInstaller) UninstallReturns(result1 error) {
	fake.uninstallMutex.Lock()
	defer fake.uninstallMutex.Unlock()
	fake.UninstallStub = nil
	fake.uninstallReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIK8sInstaller) UninstallReturnsOnCall(i int, result1 error) {
	fake.uninstallMutex.Lock()
	defer fake.uninstallMutex.Unlock()
	fake.UninstallStub = nil
	if fake.uninstallReturnsOnCall == nil {
		fake.uninstallReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.uninstallReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIK8sInstaller) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	fake.uninstallMutex.RLock()
	defer fake.uninstallMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIK8sInstaller) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
