package RtMidiGo

/*
#cgo CXXFLAGS: -g -std=c++11
#cgo LDFLAGS: -g
#cgo linux CXXFLAGS: -D__LINUX_ALSA__
#cgo linux LDFLAGS: -lasound -pthread -lrtmidi
#include <stdlib.h>
#include <stdint.h>
#include "rtmidi/rtmidi_c.h"

extern void _Callback(double ts, unsigned char *msg, size_t msgsz, void *arg);

static inline void midiInCallback(double ts, const unsigned char *msg, size_t msgsz, void *arg) {
	_Callback(ts, (unsigned char*) msg, msgsz, arg);
}

static inline void SetCallback(RtMidiPtr in, int id) {
	rtmidi_in_set_callback(in, midiInCallback, (void*)(uintptr_t) id);
}
*/
import "C"
import (
	"sync"
	"unsafe"
)

var (
	mutex sync.Mutex
	id    int
	ins   = map[int]*RtMidiIn{}
	outs  = map[int]*RtMidiOut{}
)

func ApiDisplayName(api uint32) string {
	return C.GoString(C.rtmidi_api_display_name(api))
}

func ApiName(api uint32) string {
	return C.GoString(C.rtmidi_api_name(api))
}

func CompiledApiByName(name string) uint32 {
	return C.rtmidi_compiled_api_by_name(C.CString(name))
}

func CreateIn(api uint32, clientName string, queueSizeLimit uint) *RtMidiIn {
	port := &RtMidiIn{
		ptr: C.rtmidi_in_create(api, C.CString(clientName), C.uint(queueSizeLimit)),
		id:  _NextId(),
	}

	ins[port.id] = port

	return port
}

func CreateInDefault() *RtMidiIn {
	port := &RtMidiIn{
		ptr: C.rtmidi_in_create_default(),
		id:  _NextId(),
	}

	ins[port.id] = port

	return port
}

func CreateOut(api uint32, clientName string) *RtMidiOut {
	port := &RtMidiOut{
		ptr: C.rtmidi_out_create(api, C.CString(clientName)),
		id:  _NextId(),
	}

	outs[port.id] = port

	return port
}

func CreateOutDefault() *RtMidiOut {
	port := &RtMidiOut{
		ptr: C.rtmidi_out_create_default(),
		id:  _NextId(),
	}

	outs[port.id] = port

	return port
}

func GetCompiledApi() []uint32 {
	apiCount := C.rtmidi_get_compiled_api(nil, 0)

	apis := make([]C.enum_RtMidiApi, apiCount)
	C.rtmidi_get_compiled_api(&apis[0], C.uint(apiCount))

	return apis
}

func _NextId() int {
	mutex.Lock()
	defer mutex.Unlock()

	id = id + 1

	return id
}

//export _Callback
func _Callback(ts C.double, msg *C.uchar, msgsz C.size_t, arg unsafe.Pointer) {
	id := int(uintptr(arg))
	m := ins[id]

	m.callback(m, C.GoBytes(unsafe.Pointer(msg), C.int(msgsz)), float64(ts))
}

func _Close(device C.RtMidiPtr) {
	C.rtmidi_close_port(device)
}

func _GetPortCount(device C.RtMidiPtr) int {
	return int(C.rtmidi_get_port_count(device))
}

func _GetPortName(device C.RtMidiPtr, portNumber int) string {
	bufLen := C.int(0)

	C.rtmidi_get_port_name(device, C.uint(portNumber), nil, &bufLen)

	bufOut := make([]byte, int(bufLen))
	p := (*C.char)(unsafe.Pointer(&bufOut[0]))

	C.rtmidi_get_port_name(device, C.uint(portNumber), p, &bufLen)

	return string(bufOut[0 : bufLen-1])
}

func _GetPortNames(device C.RtMidiPtr) map[int]string {
	portCount := _GetPortCount(device)
	portNames := make(map[int]string, portCount)

	for portNumber := 0; portNumber < portCount; portNumber++ {
		portNames[portNumber] = _GetPortName(device, portNumber)
	}

	return portNames
}

func _Open(device C.RtMidiPtr, portNumber int, portName string) {
	C.rtmidi_open_port(device, C.uint(portNumber), C.CString(portName))
}

func _OpenVirtual(device C.RtMidiPtr, portName string) {
	C.rtmidi_open_virtual_port(device, C.CString(portName))
}

type RtMidiInterface interface {
	Close()
	Free()
	GetCurrentApi() uint32
	GetPortCount() int
	GetPortName(portNumber int) string
	GetPortNames() []string
	Open(portNumber int, portName string)
	OpenVirtual(portName string)
}

type RtMidiIn struct {
	RtMidiInterface
	id       int
	ptr      C.RtMidiInPtr
	callback func(in *RtMidiIn, msg []byte, ts float64)
	CH       chan []byte
}

func (m *RtMidiIn) CancelCallback() {
	C.rtmidi_in_cancel_callback(m.ptr)
}

func (m *RtMidiIn) Close() {
	close(m.CH)

	_Close(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiIn) Free() {
	C.rtmidi_in_free(m.ptr)
}

func (m *RtMidiIn) GetCurrentApi() uint32 {
	return C.rtmidi_in_get_current_api(m.ptr)
}

func (m *RtMidiIn) GetMessage() []byte {
	var message *C.uchar
	var size C.size_t

	C.rtmidi_in_get_message(m.ptr, message, &size)

	return C.GoBytes(unsafe.Pointer(message), C.int(size))
}

func (m *RtMidiIn) GetPortCount() int {
	return _GetPortCount(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiIn) GetPortName(portNumber int) string {
	return _GetPortName(C.RtMidiPtr(m.ptr), portNumber)
}

func (m *RtMidiIn) GetPortNames() map[int]string {
	return _GetPortNames(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiIn) IgnoreTypes(midiSysex bool, midiTime bool, midiSense bool) {
	C.rtmidi_in_ignore_types(m.ptr, C.bool(midiSysex), C.bool(midiTime), C.bool(midiSense))
}

func (m *RtMidiIn) Open(portNumber int, portName string) {
	_Open(C.RtMidiPtr(m.ptr), portNumber, portName)
}

func (m *RtMidiIn) OpenVirtual(portName string) {
	_OpenVirtual(C.RtMidiPtr(m.ptr), portName)
}

func (m *RtMidiIn) OpenChannel(portName string) {
	m.CH = make(chan []byte)

	m.SetCallback(func(in *RtMidiIn, msg []byte, ts float64) {
		m.CH <- msg
	})
}

func (m *RtMidiIn) SetCallback(callback func(in *RtMidiIn, msg []byte, ts float64)) {
	m.callback = callback

	C.SetCallback(m.ptr, C.int(m.id))
}

type RtMidiOut struct {
	RtMidiInterface
	id  int
	ptr C.RtMidiOutPtr
	CH  chan []byte
}

func (m *RtMidiOut) Close() {
	close(m.CH)

	_Close(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiOut) Free() {
	C.rtmidi_out_free(m.ptr)
}

func (m *RtMidiOut) GetCurrentApi() uint32 {
	return C.rtmidi_out_get_current_api(m.ptr)
}

func (m *RtMidiOut) GetPortCount() int {
	return _GetPortCount(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiOut) GetPortName(portNumber int) string {
	return _GetPortName(C.RtMidiPtr(m.ptr), portNumber)
}

func (m *RtMidiOut) GetPortNames() map[int]string {
	return _GetPortNames(C.RtMidiPtr(m.ptr))
}

func (m *RtMidiOut) Open(portNumber int, portName string) {
	m.OpenChannel()

	_Open(C.RtMidiPtr(m.ptr), portNumber, portName)
}

func (m *RtMidiOut) OpenVirtual(portName string) {
	m.OpenChannel()

	_OpenVirtual(C.RtMidiPtr(m.ptr), portName)
}

func (m *RtMidiOut) OpenChannel() {
	m.CH = make(chan []byte)

	go m.LoopChannel()
}

func (m *RtMidiOut) Send(message []byte) {
	p := C.CBytes(message)
	defer C.free(unsafe.Pointer(p))

	C.rtmidi_out_send_message(m.ptr, (*C.uchar)(p), C.int(len(message)))
}

func (m *RtMidiOut) LoopChannel() {
	for {
		v, ok := <-m.CH

		if !ok {
			return
		}

		m.Send(v)
	}
}
