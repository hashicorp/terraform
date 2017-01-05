// Copyright (c) 2013 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

var registeredReceivers = make(map[string]reflect.Type)

// RegisterReceiver records a custom receiver type, identified by a value
// of that type (second argument), under the specified name. Registered
// names can be used in the "name" attribute of <custom> config items.
//
// RegisterReceiver takes the type of the receiver argument, without taking
// the value into the account. So do NOT enter any data to the second argument
// and only call it like:
//     RegisterReceiver("somename", &MyReceiverType{})
//
// After that, when a '<custom>' config tag with this name is used,
// a receiver of the specified type would be instantiated. Check
// CustomReceiver comments for interface details.
//
// NOTE 1: RegisterReceiver fails if you attempt to register different types
// with the same name.
//
// NOTE 2: RegisterReceiver registers those receivers that must be used in
// the configuration files (<custom> items). Basically it is just the way
// you tell seelog config parser what should it do when it meets a
// <custom> tag with a specific name and data attributes.
//
// But If you are only using seelog as a proxy to an already instantiated
// CustomReceiver (via LoggerFromCustomReceiver func), you should not call RegisterReceiver.
func RegisterReceiver(name string, receiver CustomReceiver) {
	newType := reflect.TypeOf(reflect.ValueOf(receiver).Elem().Interface())
	if t, ok := registeredReceivers[name]; ok && t != newType {
		panic(fmt.Sprintf("duplicate types for %s: %s != %s", name, t, newType))
	}
	registeredReceivers[name] = newType
}

func customReceiverByName(name string) (creceiver CustomReceiver, err error) {
	rt, ok := registeredReceivers[name]
	if !ok {
		return nil, fmt.Errorf("custom receiver name not registered: '%s'", name)
	}
	v, ok := reflect.New(rt).Interface().(CustomReceiver)
	if !ok {
		return nil, fmt.Errorf("cannot instantiate receiver with name='%s'", name)
	}
	return v, nil
}

// CustomReceiverInitArgs represent arguments passed to the CustomReceiver.Init
// func when custom receiver is being initialized.
type CustomReceiverInitArgs struct {
	// XmlCustomAttrs represent '<custom>' xml config item attributes that
	// start with "data-". Map keys will be the attribute names without the "data-".
	// Map values will the those attribute values.
	//
	// E.g. if you have a '<custom name="somename" data-attr1="a1" data-attr2="a2"/>'
	// you will get map with 2 key-value pairs: "attr1"->"a1", "attr2"->"a2"
	//
	// Note that in custom items you can only use allowed attributes, like "name" and
	// your custom attributes, starting with "data-". Any other will lead to a
	// parsing error.
	XmlCustomAttrs map[string]string
}

// CustomReceiver is the interface that external custom seelog message receivers
// must implement in order to be able to process seelog messages. Those receivers
// are set in the xml config file using the <custom> tag. Check receivers reference
// wiki section on that.
//
// Use seelog.RegisterReceiver on the receiver type before using it.
type CustomReceiver interface {
	// ReceiveMessage is called when the custom receiver gets seelog message from
	// a parent dispatcher.
	//
	// Message, level and context args represent all data that was included in the seelog
	// message at the time it was logged.
	//
	// The formatting is already applied to the message and depends on the config
	// like with any other receiver.
	//
	// If you would like to inform seelog of an error that happened during the handling of
	// the message, return a non-nil error. This way you'll end up seeing your error like
	// any other internal seelog error.
	ReceiveMessage(message string, level LogLevel, context LogContextInterface) error

	// AfterParse is called immediately after your custom receiver is instantiated by
	// the xml config parser. So, if you need to do any startup logic after config parsing,
	// like opening file or allocating any resources after the receiver is instantiated, do it here.
	//
	// If this func returns a non-nil error, then the loading procedure will fail. E.g.
	// if you are loading a seelog xml config, the parser would not finish the loading
	// procedure and inform about an error like with any other config error.
	//
	// If your custom logger needs some configuration, you can use custom attributes in
	// your config. Check CustomReceiverInitArgs.XmlCustomAttrs comments.
	//
	// IMPORTANT: This func is NOT called when the LoggerFromCustomReceiver func is used
	// to create seelog proxy logger using the custom receiver. This func is only called when
	// receiver is instantiated from a config.
	AfterParse(initArgs CustomReceiverInitArgs) error

	// Flush is called when the custom receiver gets a 'flush' directive from a
	// parent receiver. If custom receiver implements some kind of buffering or
	// queing, then the appropriate reaction on a flush message is synchronous
	// flushing of all those queues/buffers. If custom receiver doesn't have
	// such mechanisms, then flush implementation may be left empty.
	Flush()

	// Close is called when the custom receiver gets a 'close' directive from a
	// parent receiver. This happens when a top-level seelog dispatcher is sending
	// 'close' to all child nodes and it means that current seelog logger is being closed.
	// If you need to do any cleanup after your custom receiver is done, you should do
	// it here.
	Close() error
}

type customReceiverDispatcher struct {
	formatter          *formatter
	innerReceiver      CustomReceiver
	customReceiverName string
	usedArgs           CustomReceiverInitArgs
}

// NewCustomReceiverDispatcher creates a customReceiverDispatcher which dispatches data to a specific receiver created
// using a <custom> tag in the config file.
func NewCustomReceiverDispatcher(formatter *formatter, customReceiverName string, cArgs CustomReceiverInitArgs) (*customReceiverDispatcher, error) {
	if formatter == nil {
		return nil, errors.New("formatter cannot be nil")
	}
	if len(customReceiverName) == 0 {
		return nil, errors.New("custom receiver name cannot be empty")
	}

	creceiver, err := customReceiverByName(customReceiverName)
	if err != nil {
		return nil, err
	}
	err = creceiver.AfterParse(cArgs)
	if err != nil {
		return nil, err
	}
	disp := &customReceiverDispatcher{formatter, creceiver, customReceiverName, cArgs}

	return disp, nil
}

// NewCustomReceiverDispatcherByValue is basically the same as NewCustomReceiverDispatcher, but using
// a specific CustomReceiver value instead of instantiating a new one by type.
func NewCustomReceiverDispatcherByValue(formatter *formatter, customReceiver CustomReceiver, name string, cArgs CustomReceiverInitArgs) (*customReceiverDispatcher, error) {
	if formatter == nil {
		return nil, errors.New("formatter cannot be nil")
	}
	if customReceiver == nil {
		return nil, errors.New("customReceiver cannot be nil")
	}
	disp := &customReceiverDispatcher{formatter, customReceiver, name, cArgs}

	return disp, nil
}

// CustomReceiver implementation. Check CustomReceiver comments.
func (disp *customReceiverDispatcher) Dispatch(
	message string,
	level LogLevel,
	context LogContextInterface,
	errorFunc func(err error)) {

	defer func() {
		if err := recover(); err != nil {
			errorFunc(fmt.Errorf("panic in custom receiver '%s'.Dispatch: %s", reflect.TypeOf(disp.innerReceiver), err))
		}
	}()

	err := disp.innerReceiver.ReceiveMessage(disp.formatter.Format(message, level, context), level, context)
	if err != nil {
		errorFunc(err)
	}
}

// CustomReceiver implementation. Check CustomReceiver comments.
func (disp *customReceiverDispatcher) Flush() {
	disp.innerReceiver.Flush()
}

// CustomReceiver implementation. Check CustomReceiver comments.
func (disp *customReceiverDispatcher) Close() error {
	disp.innerReceiver.Flush()

	err := disp.innerReceiver.Close()
	if err != nil {
		return err
	}

	return nil
}

func (disp *customReceiverDispatcher) String() string {
	datas := ""
	skeys := make([]string, 0, len(disp.usedArgs.XmlCustomAttrs))
	for i := range disp.usedArgs.XmlCustomAttrs {
		skeys = append(skeys, i)
	}
	sort.Strings(skeys)
	for _, key := range skeys {
		datas += fmt.Sprintf("<%s, %s> ", key, disp.usedArgs.XmlCustomAttrs[key])
	}

	str := fmt.Sprintf("Custom receiver %s [fmt='%s'],[data='%s'],[inner='%s']\n",
		disp.customReceiverName, disp.formatter.String(), datas, disp.innerReceiver)

	return str
}
