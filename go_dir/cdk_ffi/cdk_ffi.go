package cdk_ffi

// #include <cdk_ffi.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// This is needed, because as of go 1.24
// type RustBuffer C.RustBuffer cannot have methods,
// RustBuffer is treated as non-local type
type GoRustBuffer struct {
	inner C.RustBuffer
}

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() uint64
	Capacity() uint64
}

func RustBufferFromExternal(b RustBufferI) GoRustBuffer {
	return GoRustBuffer{
		inner: C.RustBuffer{
			capacity: C.uint64_t(b.Capacity()),
			len:      C.uint64_t(b.Len()),
			data:     (*C.uchar)(b.Data()),
		},
	}
}

func (cb GoRustBuffer) Capacity() uint64 {
	return uint64(cb.inner.capacity)
}

func (cb GoRustBuffer) Len() uint64 {
	return uint64(cb.inner.len)
}

func (cb GoRustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.inner.data)
}

func (cb GoRustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.inner.data), C.uint64_t(cb.inner.len))
	return bytes.NewReader(b)
}

func (cb GoRustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_cdk_ffi_rustbuffer_free(cb.inner, status)
		return false
	})
}

func (cb GoRustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.inner.data), C.int(cb.inner.len))
}

func stringToRustBuffer(str string) C.RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) C.RustBuffer {
	if len(b) == 0 {
		return C.RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) C.RustBuffer {
		return C.ffi_cdk_ffi_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) C.RustBuffer
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) C.RustBuffer {
	// This might be not the most efficient way but it does not require knowing allocation size
	// beforehand
	var buffer bytes.Buffer
	bufWriter.Write(&buffer, value)

	bytes, err := io.ReadAll(&buffer)
	if err != nil {
		panic(fmt.Errorf("reading written data: %w", err))
	}
	return bytesToRustBuffer(bytes)
}

func LiftFromRustBuffer[GoType any](bufReader BufReader[GoType], rbuf RustBufferI) GoType {
	defer rbuf.Free()
	reader := rbuf.AsReader()
	item := bufReader.Read(reader)
	if reader.Len() > 0 {
		// TODO: Remove this
		leftover, _ := io.ReadAll(reader)
		panic(fmt.Errorf("Junk remaining in buffer after lifting: %s", string(leftover)))
	}
	return item
}

func rustCallWithError[E any, U any](converter BufReader[*E], callback func(*C.RustCallStatus) U) (U, *E) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)
	return returnValue, err
}

func checkCallStatus[E any](converter BufReader[*E], status C.RustCallStatus) *E {
	switch status.code {
	case 0:
		return nil
	case 1:
		return LiftFromRustBuffer(converter, GoRustBuffer{inner: status.errorBuf})
	case 2:
		// when the rust code sees a panic, it tries to construct a rustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{inner: status.errorBuf})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		panic(fmt.Errorf("unknown status code: %d", status.code))
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a C.RustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: status.errorBuf,
			})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError[error](nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
}

type NativeError interface {
	AsError() error
}

func writeInt8(writer io.Writer, value int8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint8(writer io.Writer, value uint8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt16(writer io.Writer, value int16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint16(writer io.Writer, value uint16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt32(writer io.Writer, value int32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint32(writer io.Writer, value uint32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt64(writer io.Writer, value int64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint64(writer io.Writer, value uint64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat32(writer io.Writer, value float32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat64(writer io.Writer, value float64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func readInt8(reader io.Reader) int8 {
	var result int8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint8(reader io.Reader) uint8 {
	var result uint8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt16(reader io.Reader) int16 {
	var result int16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint16(reader io.Reader) uint16 {
	var result uint16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt32(reader io.Reader) int32 {
	var result int32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint32(reader io.Reader) uint32 {
	var result uint32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt64(reader io.Reader) int64 {
	var result int64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint64(reader io.Reader) uint64 {
	var result uint64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat32(reader io.Reader) float32 {
	var result float32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat64(reader io.Reader) float64 {
	var result float64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func init() {

	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 26
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_cdk_ffi_uniffi_contract_version()
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("cdk_ffi: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_func_generate_mnemonic()
		})
		if checksum != 44815 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_func_generate_mnemonic: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_balance()
		})
		if checksum != 40463 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_balance: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_get_mint_info()
		})
		if checksum != 13159 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_get_mint_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_melt()
		})
		if checksum != 3275 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_melt: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_melt_quote()
		})
		if checksum != 39876 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_melt_quote: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_mint()
		})
		if checksum != 58480 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_mint: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_mint_quote()
		})
		if checksum != 42885 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_mint_quote: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_mint_quote_state()
		})
		if checksum != 60165 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_mint_quote_state: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_mint_url()
		})
		if checksum != 18647 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_mint_url: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_prepare_send()
		})
		if checksum != 46706 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_prepare_send: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_send()
		})
		if checksum != 15473 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_send: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_method_ffiwallet_unit()
		})
		if checksum != 4593 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_method_ffiwallet_unit: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_constructor_ffilocalstore_new()
		})
		if checksum != 15364 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_constructor_ffilocalstore_new: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_constructor_ffilocalstore_new_with_path()
		})
		if checksum != 766 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_constructor_ffilocalstore_new_with_path: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_constructor_ffiwallet_from_mnemonic()
		})
		if checksum != 63545 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_constructor_ffiwallet_from_mnemonic: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_cdk_ffi_checksum_constructor_ffiwallet_restore_from_mnemonic()
		})
		if checksum != 38466 {
			// If this happens try cleaning and rebuilding your project
			panic("cdk_ffi: uniffi_cdk_ffi_checksum_constructor_ffiwallet_restore_from_mnemonic: UniFFI API checksum mismatch")
		}
	}
}

type FfiConverterUint64 struct{}

var FfiConverterUint64INSTANCE = FfiConverterUint64{}

func (FfiConverterUint64) Lower(value uint64) C.uint64_t {
	return C.uint64_t(value)
}

func (FfiConverterUint64) Write(writer io.Writer, value uint64) {
	writeUint64(writer, value)
}

func (FfiConverterUint64) Lift(value C.uint64_t) uint64 {
	return uint64(value)
}

func (FfiConverterUint64) Read(reader io.Reader) uint64 {
	return readUint64(reader)
}

type FfiDestroyerUint64 struct{}

func (FfiDestroyerUint64) Destroy(_ uint64) {}

type FfiConverterBool struct{}

var FfiConverterBoolINSTANCE = FfiConverterBool{}

func (FfiConverterBool) Lower(value bool) C.int8_t {
	if value {
		return C.int8_t(1)
	}
	return C.int8_t(0)
}

func (FfiConverterBool) Write(writer io.Writer, value bool) {
	if value {
		writeInt8(writer, 1)
	} else {
		writeInt8(writer, 0)
	}
}

func (FfiConverterBool) Lift(value C.int8_t) bool {
	return value != 0
}

func (FfiConverterBool) Read(reader io.Reader) bool {
	return readInt8(reader) != 0
}

type FfiDestroyerBool struct{}

func (FfiDestroyerBool) Destroy(_ bool) {}

type FfiConverterString struct{}

var FfiConverterStringINSTANCE = FfiConverterString{}

func (FfiConverterString) Lift(rb RustBufferI) string {
	defer rb.Free()
	reader := rb.AsReader()
	b, err := io.ReadAll(reader)
	if err != nil {
		panic(fmt.Errorf("reading reader: %w", err))
	}
	return string(b)
}

func (FfiConverterString) Read(reader io.Reader) string {
	length := readInt32(reader)
	buffer := make([]byte, length)
	read_length, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		panic(err)
	}
	if read_length != int(length) {
		panic(fmt.Errorf("bad read length when reading string, expected %d, read %d", length, read_length))
	}
	return string(buffer)
}

func (FfiConverterString) Lower(value string) C.RustBuffer {
	return stringToRustBuffer(value)
}

func (FfiConverterString) Write(writer io.Writer, value string) {
	if len(value) > math.MaxInt32 {
		panic("String is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	write_length, err := io.WriteString(writer, value)
	if err != nil {
		panic(err)
	}
	if write_length != len(value) {
		panic(fmt.Errorf("bad write length when writing string, expected %d, written %d", len(value), write_length))
	}
}

type FfiDestroyerString struct{}

func (FfiDestroyerString) Destroy(_ string) {}

// Below is an implementation of synchronization requirements outlined in the link.
// https://github.com/mozilla/uniffi-rs/blob/0dc031132d9493ca812c3af6e7dd60ad2ea95bf0/uniffi_bindgen/src/bindings/kotlin/templates/ObjectRuntime.kt#L31

type FfiObject struct {
	pointer       unsafe.Pointer
	callCounter   atomic.Int64
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer
	freeFunction  func(unsafe.Pointer, *C.RustCallStatus)
	destroyed     atomic.Bool
}

func newFfiObject(
	pointer unsafe.Pointer,
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer,
	freeFunction func(unsafe.Pointer, *C.RustCallStatus),
) FfiObject {
	return FfiObject{
		pointer:       pointer,
		cloneFunction: cloneFunction,
		freeFunction:  freeFunction,
	}
}

func (ffiObject *FfiObject) incrementPointer(debugName string) unsafe.Pointer {
	for {
		counter := ffiObject.callCounter.Load()
		if counter <= -1 {
			panic(fmt.Errorf("%v object has already been destroyed", debugName))
		}
		if counter == math.MaxInt64 {
			panic(fmt.Errorf("%v object call counter would overflow", debugName))
		}
		if ffiObject.callCounter.CompareAndSwap(counter, counter+1) {
			break
		}
	}

	return rustCall(func(status *C.RustCallStatus) unsafe.Pointer {
		return ffiObject.cloneFunction(ffiObject.pointer, status)
	})
}

func (ffiObject *FfiObject) decrementPointer() {
	if ffiObject.callCounter.Add(-1) == -1 {
		ffiObject.freeRustArcPtr()
	}
}

func (ffiObject *FfiObject) destroy() {
	if ffiObject.destroyed.CompareAndSwap(false, true) {
		if ffiObject.callCounter.Add(-1) == -1 {
			ffiObject.freeRustArcPtr()
		}
	}
}

func (ffiObject *FfiObject) freeRustArcPtr() {
	rustCall(func(status *C.RustCallStatus) int32 {
		ffiObject.freeFunction(ffiObject.pointer, status)
		return 0
	})
}

type FfiLocalStoreInterface interface {
}
type FfiLocalStore struct {
	ffiObject FfiObject
}

func NewFfiLocalStore() (*FfiLocalStore, error) {
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_cdk_ffi_fn_constructor_ffilocalstore_new(_uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *FfiLocalStore
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiLocalStoreINSTANCE.Lift(_uniffiRV), nil
	}
}

func FfiLocalStoreNewWithPath(dbPath *string) (*FfiLocalStore, error) {
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_cdk_ffi_fn_constructor_ffilocalstore_new_with_path(FfiConverterOptionalStringINSTANCE.Lower(dbPath), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *FfiLocalStore
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiLocalStoreINSTANCE.Lift(_uniffiRV), nil
	}
}

func (object *FfiLocalStore) Destroy() {
	runtime.SetFinalizer(object, nil)
	object.ffiObject.destroy()
}

type FfiConverterFfiLocalStore struct{}

var FfiConverterFfiLocalStoreINSTANCE = FfiConverterFfiLocalStore{}

func (c FfiConverterFfiLocalStore) Lift(pointer unsafe.Pointer) *FfiLocalStore {
	result := &FfiLocalStore{
		newFfiObject(
			pointer,
			func(pointer unsafe.Pointer, status *C.RustCallStatus) unsafe.Pointer {
				return C.uniffi_cdk_ffi_fn_clone_ffilocalstore(pointer, status)
			},
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_cdk_ffi_fn_free_ffilocalstore(pointer, status)
			},
		),
	}
	runtime.SetFinalizer(result, (*FfiLocalStore).Destroy)
	return result
}

func (c FfiConverterFfiLocalStore) Read(reader io.Reader) *FfiLocalStore {
	return c.Lift(unsafe.Pointer(uintptr(readUint64(reader))))
}

func (c FfiConverterFfiLocalStore) Lower(value *FfiLocalStore) unsafe.Pointer {
	// TODO: this is bad - all synchronization from ObjectRuntime.go is discarded here,
	// because the pointer will be decremented immediately after this function returns,
	// and someone will be left holding onto a non-locked pointer.
	pointer := value.ffiObject.incrementPointer("*FfiLocalStore")
	defer value.ffiObject.decrementPointer()
	return pointer

}

func (c FfiConverterFfiLocalStore) Write(writer io.Writer, value *FfiLocalStore) {
	writeUint64(writer, uint64(uintptr(c.Lower(value))))
}

type FfiDestroyerFfiLocalStore struct{}

func (_ FfiDestroyerFfiLocalStore) Destroy(value *FfiLocalStore) {
	value.Destroy()
}

type FfiWalletInterface interface {
	Balance() (FfiAmount, error)
	// Fetch and initialize mint information
	// This should be called after wallet creation to set up the mint in the database
	GetMintInfo() (string, error)
	// Execute a melt operation (pay Lightning invoice)
	Melt(quoteId string) (FfiMelted, error)
	// Create a melt quote for paying a Lightning invoice
	MeltQuote(request string) (FfiMeltQuote, error)
	Mint(quoteId string, splitTarget FfiSplitTarget) (FfiAmount, error)
	MintQuote(amount FfiAmount, description *string) (FfiMintQuote, error)
	MintQuoteState(quoteId string) (FfiMintQuoteBolt11Response, error)
	MintUrl() string
	PrepareSend(amount FfiAmount, options FfiSendOptions) (FfiPreparedSend, error)
	Send(amount FfiAmount, options FfiSendOptions, memo *FfiSendMemo) (FfiToken, error)
	Unit() string
}
type FfiWallet struct {
	ffiObject FfiObject
}

func FfiWalletFromMnemonic(mintUrl string, unit FfiCurrencyUnit, localstore *FfiLocalStore, mnemonicWords string) (*FfiWallet, error) {
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_cdk_ffi_fn_constructor_ffiwallet_from_mnemonic(FfiConverterStringINSTANCE.Lower(mintUrl), FfiConverterFfiCurrencyUnitINSTANCE.Lower(unit), FfiConverterFfiLocalStoreINSTANCE.Lower(localstore), FfiConverterStringINSTANCE.Lower(mnemonicWords), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *FfiWallet
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiWalletINSTANCE.Lift(_uniffiRV), nil
	}
}

func FfiWalletRestoreFromMnemonic(mintUrl string, unit FfiCurrencyUnit, localstore *FfiLocalStore, mnemonicWords string) (*FfiWallet, error) {
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_cdk_ffi_fn_constructor_ffiwallet_restore_from_mnemonic(FfiConverterStringINSTANCE.Lower(mintUrl), FfiConverterFfiCurrencyUnitINSTANCE.Lower(unit), FfiConverterFfiLocalStoreINSTANCE.Lower(localstore), FfiConverterStringINSTANCE.Lower(mnemonicWords), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *FfiWallet
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiWalletINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) Balance() (FfiAmount, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_balance(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiAmount
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiAmountINSTANCE.Lift(_uniffiRV), nil
	}
}

// Fetch and initialize mint information
// This should be called after wallet creation to set up the mint in the database
func (_self *FfiWallet) GetMintInfo() (string, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_get_mint_info(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), nil
	}
}

// Execute a melt operation (pay Lightning invoice)
func (_self *FfiWallet) Melt(quoteId string) (FfiMelted, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_melt(
				_pointer, FfiConverterStringINSTANCE.Lower(quoteId), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiMelted
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiMeltedINSTANCE.Lift(_uniffiRV), nil
	}
}

// Create a melt quote for paying a Lightning invoice
func (_self *FfiWallet) MeltQuote(request string) (FfiMeltQuote, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_melt_quote(
				_pointer, FfiConverterStringINSTANCE.Lower(request), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiMeltQuote
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiMeltQuoteINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) Mint(quoteId string, splitTarget FfiSplitTarget) (FfiAmount, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_mint(
				_pointer, FfiConverterStringINSTANCE.Lower(quoteId), FfiConverterFfiSplitTargetINSTANCE.Lower(splitTarget), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiAmount
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiAmountINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) MintQuote(amount FfiAmount, description *string) (FfiMintQuote, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_mint_quote(
				_pointer, FfiConverterFfiAmountINSTANCE.Lower(amount), FfiConverterOptionalStringINSTANCE.Lower(description), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiMintQuote
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiMintQuoteINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) MintQuoteState(quoteId string) (FfiMintQuoteBolt11Response, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_mint_quote_state(
				_pointer, FfiConverterStringINSTANCE.Lower(quoteId), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiMintQuoteBolt11Response
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiMintQuoteBolt11ResponseINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) MintUrl() string {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	return FfiConverterStringINSTANCE.Lift(rustCall(func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_mint_url(
				_pointer, _uniffiStatus),
		}
	}))
}

func (_self *FfiWallet) PrepareSend(amount FfiAmount, options FfiSendOptions) (FfiPreparedSend, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_prepare_send(
				_pointer, FfiConverterFfiAmountINSTANCE.Lower(amount), FfiConverterFfiSendOptionsINSTANCE.Lower(options), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiPreparedSend
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiPreparedSendINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) Send(amount FfiAmount, options FfiSendOptions, memo *FfiSendMemo) (FfiToken, error) {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_send(
				_pointer, FfiConverterFfiAmountINSTANCE.Lower(amount), FfiConverterFfiSendOptionsINSTANCE.Lower(options), FfiConverterOptionalFfiSendMemoINSTANCE.Lower(memo), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FfiToken
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFfiTokenINSTANCE.Lift(_uniffiRV), nil
	}
}

func (_self *FfiWallet) Unit() string {
	_pointer := _self.ffiObject.incrementPointer("*FfiWallet")
	defer _self.ffiObject.decrementPointer()
	return FfiConverterStringINSTANCE.Lift(rustCall(func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_method_ffiwallet_unit(
				_pointer, _uniffiStatus),
		}
	}))
}
func (object *FfiWallet) Destroy() {
	runtime.SetFinalizer(object, nil)
	object.ffiObject.destroy()
}

type FfiConverterFfiWallet struct{}

var FfiConverterFfiWalletINSTANCE = FfiConverterFfiWallet{}

func (c FfiConverterFfiWallet) Lift(pointer unsafe.Pointer) *FfiWallet {
	result := &FfiWallet{
		newFfiObject(
			pointer,
			func(pointer unsafe.Pointer, status *C.RustCallStatus) unsafe.Pointer {
				return C.uniffi_cdk_ffi_fn_clone_ffiwallet(pointer, status)
			},
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_cdk_ffi_fn_free_ffiwallet(pointer, status)
			},
		),
	}
	runtime.SetFinalizer(result, (*FfiWallet).Destroy)
	return result
}

func (c FfiConverterFfiWallet) Read(reader io.Reader) *FfiWallet {
	return c.Lift(unsafe.Pointer(uintptr(readUint64(reader))))
}

func (c FfiConverterFfiWallet) Lower(value *FfiWallet) unsafe.Pointer {
	// TODO: this is bad - all synchronization from ObjectRuntime.go is discarded here,
	// because the pointer will be decremented immediately after this function returns,
	// and someone will be left holding onto a non-locked pointer.
	pointer := value.ffiObject.incrementPointer("*FfiWallet")
	defer value.ffiObject.decrementPointer()
	return pointer

}

func (c FfiConverterFfiWallet) Write(writer io.Writer, value *FfiWallet) {
	writeUint64(writer, uint64(uintptr(c.Lower(value))))
}

type FfiDestroyerFfiWallet struct{}

func (_ FfiDestroyerFfiWallet) Destroy(value *FfiWallet) {
	value.Destroy()
}

type FfiAmount struct {
	Value uint64
}

func (r *FfiAmount) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.Value)
}

type FfiConverterFfiAmount struct{}

var FfiConverterFfiAmountINSTANCE = FfiConverterFfiAmount{}

func (c FfiConverterFfiAmount) Lift(rb RustBufferI) FfiAmount {
	return LiftFromRustBuffer[FfiAmount](c, rb)
}

func (c FfiConverterFfiAmount) Read(reader io.Reader) FfiAmount {
	return FfiAmount{
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiAmount) Lower(value FfiAmount) C.RustBuffer {
	return LowerIntoRustBuffer[FfiAmount](c, value)
}

func (c FfiConverterFfiAmount) Write(writer io.Writer, value FfiAmount) {
	FfiConverterUint64INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerFfiAmount struct{}

func (_ FfiDestroyerFfiAmount) Destroy(value FfiAmount) {
	value.Destroy()
}

type FfiMeltQuote struct {
	Id              string
	Unit            string
	Amount          FfiAmount
	Request         string
	FeeReserve      FfiAmount
	Expiry          uint64
	PaymentPreimage *string
}

func (r *FfiMeltQuote) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerString{}.Destroy(r.Unit)
	FfiDestroyerFfiAmount{}.Destroy(r.Amount)
	FfiDestroyerString{}.Destroy(r.Request)
	FfiDestroyerFfiAmount{}.Destroy(r.FeeReserve)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentPreimage)
}

type FfiConverterFfiMeltQuote struct{}

var FfiConverterFfiMeltQuoteINSTANCE = FfiConverterFfiMeltQuote{}

func (c FfiConverterFfiMeltQuote) Lift(rb RustBufferI) FfiMeltQuote {
	return LiftFromRustBuffer[FfiMeltQuote](c, rb)
}

func (c FfiConverterFfiMeltQuote) Read(reader io.Reader) FfiMeltQuote {
	return FfiMeltQuote{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiMeltQuote) Lower(value FfiMeltQuote) C.RustBuffer {
	return LowerIntoRustBuffer[FfiMeltQuote](c, value)
}

func (c FfiConverterFfiMeltQuote) Write(writer io.Writer, value FfiMeltQuote) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.Unit)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterStringINSTANCE.Write(writer, value.Request)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.FeeReserve)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentPreimage)
}

type FfiDestroyerFfiMeltQuote struct{}

func (_ FfiDestroyerFfiMeltQuote) Destroy(value FfiMeltQuote) {
	value.Destroy()
}

type FfiMelted struct {
	State    string
	Preimage *string
	Amount   FfiAmount
	FeePaid  FfiAmount
}

func (r *FfiMelted) Destroy() {
	FfiDestroyerString{}.Destroy(r.State)
	FfiDestroyerOptionalString{}.Destroy(r.Preimage)
	FfiDestroyerFfiAmount{}.Destroy(r.Amount)
	FfiDestroyerFfiAmount{}.Destroy(r.FeePaid)
}

type FfiConverterFfiMelted struct{}

var FfiConverterFfiMeltedINSTANCE = FfiConverterFfiMelted{}

func (c FfiConverterFfiMelted) Lift(rb RustBufferI) FfiMelted {
	return LiftFromRustBuffer[FfiMelted](c, rb)
}

func (c FfiConverterFfiMelted) Read(reader io.Reader) FfiMelted {
	return FfiMelted{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiMelted) Lower(value FfiMelted) C.RustBuffer {
	return LowerIntoRustBuffer[FfiMelted](c, value)
}

func (c FfiConverterFfiMelted) Write(writer io.Writer, value FfiMelted) {
	FfiConverterStringINSTANCE.Write(writer, value.State)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Preimage)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.FeePaid)
}

type FfiDestroyerFfiMelted struct{}

func (_ FfiDestroyerFfiMelted) Destroy(value FfiMelted) {
	value.Destroy()
}

type FfiMintQuote struct {
	Id      string
	MintUrl string
	Amount  FfiAmount
	Unit    string
	Request string
	State   FfiMintQuoteState
	Expiry  uint64
}

func (r *FfiMintQuote) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerString{}.Destroy(r.MintUrl)
	FfiDestroyerFfiAmount{}.Destroy(r.Amount)
	FfiDestroyerString{}.Destroy(r.Unit)
	FfiDestroyerString{}.Destroy(r.Request)
	FfiDestroyerFfiMintQuoteState{}.Destroy(r.State)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
}

type FfiConverterFfiMintQuote struct{}

var FfiConverterFfiMintQuoteINSTANCE = FfiConverterFfiMintQuote{}

func (c FfiConverterFfiMintQuote) Lift(rb RustBufferI) FfiMintQuote {
	return LiftFromRustBuffer[FfiMintQuote](c, rb)
}

func (c FfiConverterFfiMintQuote) Read(reader io.Reader) FfiMintQuote {
	return FfiMintQuote{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFfiMintQuoteStateINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiMintQuote) Lower(value FfiMintQuote) C.RustBuffer {
	return LowerIntoRustBuffer[FfiMintQuote](c, value)
}

func (c FfiConverterFfiMintQuote) Write(writer io.Writer, value FfiMintQuote) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.MintUrl)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterStringINSTANCE.Write(writer, value.Unit)
	FfiConverterStringINSTANCE.Write(writer, value.Request)
	FfiConverterFfiMintQuoteStateINSTANCE.Write(writer, value.State)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
}

type FfiDestroyerFfiMintQuote struct{}

func (_ FfiDestroyerFfiMintQuote) Destroy(value FfiMintQuote) {
	value.Destroy()
}

type FfiMintQuoteBolt11Response struct {
	Quote   string
	Request string
	State   FfiMintQuoteState
	Expiry  *uint64
}

func (r *FfiMintQuoteBolt11Response) Destroy() {
	FfiDestroyerString{}.Destroy(r.Quote)
	FfiDestroyerString{}.Destroy(r.Request)
	FfiDestroyerFfiMintQuoteState{}.Destroy(r.State)
	FfiDestroyerOptionalUint64{}.Destroy(r.Expiry)
}

type FfiConverterFfiMintQuoteBolt11Response struct{}

var FfiConverterFfiMintQuoteBolt11ResponseINSTANCE = FfiConverterFfiMintQuoteBolt11Response{}

func (c FfiConverterFfiMintQuoteBolt11Response) Lift(rb RustBufferI) FfiMintQuoteBolt11Response {
	return LiftFromRustBuffer[FfiMintQuoteBolt11Response](c, rb)
}

func (c FfiConverterFfiMintQuoteBolt11Response) Read(reader io.Reader) FfiMintQuoteBolt11Response {
	return FfiMintQuoteBolt11Response{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFfiMintQuoteStateINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiMintQuoteBolt11Response) Lower(value FfiMintQuoteBolt11Response) C.RustBuffer {
	return LowerIntoRustBuffer[FfiMintQuoteBolt11Response](c, value)
}

func (c FfiConverterFfiMintQuoteBolt11Response) Write(writer io.Writer, value FfiMintQuoteBolt11Response) {
	FfiConverterStringINSTANCE.Write(writer, value.Quote)
	FfiConverterStringINSTANCE.Write(writer, value.Request)
	FfiConverterFfiMintQuoteStateINSTANCE.Write(writer, value.State)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.Expiry)
}

type FfiDestroyerFfiMintQuoteBolt11Response struct{}

func (_ FfiDestroyerFfiMintQuoteBolt11Response) Destroy(value FfiMintQuoteBolt11Response) {
	value.Destroy()
}

type FfiPreparedSend struct {
	Amount   FfiAmount
	SwapFee  FfiAmount
	SendFee  FfiAmount
	TotalFee FfiAmount
}

func (r *FfiPreparedSend) Destroy() {
	FfiDestroyerFfiAmount{}.Destroy(r.Amount)
	FfiDestroyerFfiAmount{}.Destroy(r.SwapFee)
	FfiDestroyerFfiAmount{}.Destroy(r.SendFee)
	FfiDestroyerFfiAmount{}.Destroy(r.TotalFee)
}

type FfiConverterFfiPreparedSend struct{}

var FfiConverterFfiPreparedSendINSTANCE = FfiConverterFfiPreparedSend{}

func (c FfiConverterFfiPreparedSend) Lift(rb RustBufferI) FfiPreparedSend {
	return LiftFromRustBuffer[FfiPreparedSend](c, rb)
}

func (c FfiConverterFfiPreparedSend) Read(reader io.Reader) FfiPreparedSend {
	return FfiPreparedSend{
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
		FfiConverterFfiAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiPreparedSend) Lower(value FfiPreparedSend) C.RustBuffer {
	return LowerIntoRustBuffer[FfiPreparedSend](c, value)
}

func (c FfiConverterFfiPreparedSend) Write(writer io.Writer, value FfiPreparedSend) {
	FfiConverterFfiAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.SwapFee)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.SendFee)
	FfiConverterFfiAmountINSTANCE.Write(writer, value.TotalFee)
}

type FfiDestroyerFfiPreparedSend struct{}

func (_ FfiDestroyerFfiPreparedSend) Destroy(value FfiPreparedSend) {
	value.Destroy()
}

type FfiSendMemo struct {
	Memo        string
	IncludeMemo bool
}

func (r *FfiSendMemo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Memo)
	FfiDestroyerBool{}.Destroy(r.IncludeMemo)
}

type FfiConverterFfiSendMemo struct{}

var FfiConverterFfiSendMemoINSTANCE = FfiConverterFfiSendMemo{}

func (c FfiConverterFfiSendMemo) Lift(rb RustBufferI) FfiSendMemo {
	return LiftFromRustBuffer[FfiSendMemo](c, rb)
}

func (c FfiConverterFfiSendMemo) Read(reader io.Reader) FfiSendMemo {
	return FfiSendMemo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiSendMemo) Lower(value FfiSendMemo) C.RustBuffer {
	return LowerIntoRustBuffer[FfiSendMemo](c, value)
}

func (c FfiConverterFfiSendMemo) Write(writer io.Writer, value FfiSendMemo) {
	FfiConverterStringINSTANCE.Write(writer, value.Memo)
	FfiConverterBoolINSTANCE.Write(writer, value.IncludeMemo)
}

type FfiDestroyerFfiSendMemo struct{}

func (_ FfiDestroyerFfiSendMemo) Destroy(value FfiSendMemo) {
	value.Destroy()
}

type FfiSendOptions struct {
	Memo              *FfiSendMemo
	AmountSplitTarget FfiSplitTarget
	SendKind          FfiSendKind
	IncludeFee        bool
	Metadata          map[string]string
	MaxProofs         *uint64
}

func (r *FfiSendOptions) Destroy() {
	FfiDestroyerOptionalFfiSendMemo{}.Destroy(r.Memo)
	FfiDestroyerFfiSplitTarget{}.Destroy(r.AmountSplitTarget)
	FfiDestroyerFfiSendKind{}.Destroy(r.SendKind)
	FfiDestroyerBool{}.Destroy(r.IncludeFee)
	FfiDestroyerMapStringString{}.Destroy(r.Metadata)
	FfiDestroyerOptionalUint64{}.Destroy(r.MaxProofs)
}

type FfiConverterFfiSendOptions struct{}

var FfiConverterFfiSendOptionsINSTANCE = FfiConverterFfiSendOptions{}

func (c FfiConverterFfiSendOptions) Lift(rb RustBufferI) FfiSendOptions {
	return LiftFromRustBuffer[FfiSendOptions](c, rb)
}

func (c FfiConverterFfiSendOptions) Read(reader io.Reader) FfiSendOptions {
	return FfiSendOptions{
		FfiConverterOptionalFfiSendMemoINSTANCE.Read(reader),
		FfiConverterFfiSplitTargetINSTANCE.Read(reader),
		FfiConverterFfiSendKindINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterMapStringStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiSendOptions) Lower(value FfiSendOptions) C.RustBuffer {
	return LowerIntoRustBuffer[FfiSendOptions](c, value)
}

func (c FfiConverterFfiSendOptions) Write(writer io.Writer, value FfiSendOptions) {
	FfiConverterOptionalFfiSendMemoINSTANCE.Write(writer, value.Memo)
	FfiConverterFfiSplitTargetINSTANCE.Write(writer, value.AmountSplitTarget)
	FfiConverterFfiSendKindINSTANCE.Write(writer, value.SendKind)
	FfiConverterBoolINSTANCE.Write(writer, value.IncludeFee)
	FfiConverterMapStringStringINSTANCE.Write(writer, value.Metadata)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.MaxProofs)
}

type FfiDestroyerFfiSendOptions struct{}

func (_ FfiDestroyerFfiSendOptions) Destroy(value FfiSendOptions) {
	value.Destroy()
}

type FfiToken struct {
	TokenString string
	Mint        string
	Memo        *string
	Unit        string
}

func (r *FfiToken) Destroy() {
	FfiDestroyerString{}.Destroy(r.TokenString)
	FfiDestroyerString{}.Destroy(r.Mint)
	FfiDestroyerOptionalString{}.Destroy(r.Memo)
	FfiDestroyerString{}.Destroy(r.Unit)
}

type FfiConverterFfiToken struct{}

var FfiConverterFfiTokenINSTANCE = FfiConverterFfiToken{}

func (c FfiConverterFfiToken) Lift(rb RustBufferI) FfiToken {
	return LiftFromRustBuffer[FfiToken](c, rb)
}

func (c FfiConverterFfiToken) Read(reader io.Reader) FfiToken {
	return FfiToken{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterFfiToken) Lower(value FfiToken) C.RustBuffer {
	return LowerIntoRustBuffer[FfiToken](c, value)
}

func (c FfiConverterFfiToken) Write(writer io.Writer, value FfiToken) {
	FfiConverterStringINSTANCE.Write(writer, value.TokenString)
	FfiConverterStringINSTANCE.Write(writer, value.Mint)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Memo)
	FfiConverterStringINSTANCE.Write(writer, value.Unit)
}

type FfiDestroyerFfiToken struct{}

func (_ FfiDestroyerFfiToken) Destroy(value FfiToken) {
	value.Destroy()
}

type FfiCurrencyUnit uint

const (
	FfiCurrencyUnitSat  FfiCurrencyUnit = 1
	FfiCurrencyUnitMsat FfiCurrencyUnit = 2
	FfiCurrencyUnitUsd  FfiCurrencyUnit = 3
	FfiCurrencyUnitEur  FfiCurrencyUnit = 4
)

type FfiConverterFfiCurrencyUnit struct{}

var FfiConverterFfiCurrencyUnitINSTANCE = FfiConverterFfiCurrencyUnit{}

func (c FfiConverterFfiCurrencyUnit) Lift(rb RustBufferI) FfiCurrencyUnit {
	return LiftFromRustBuffer[FfiCurrencyUnit](c, rb)
}

func (c FfiConverterFfiCurrencyUnit) Lower(value FfiCurrencyUnit) C.RustBuffer {
	return LowerIntoRustBuffer[FfiCurrencyUnit](c, value)
}
func (FfiConverterFfiCurrencyUnit) Read(reader io.Reader) FfiCurrencyUnit {
	id := readInt32(reader)
	return FfiCurrencyUnit(id)
}

func (FfiConverterFfiCurrencyUnit) Write(writer io.Writer, value FfiCurrencyUnit) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerFfiCurrencyUnit struct{}

func (_ FfiDestroyerFfiCurrencyUnit) Destroy(value FfiCurrencyUnit) {
}

type FfiError struct {
	err error
}

// Convience method to turn *FfiError into error
// Avoiding treating nil pointer as non nil error interface
func (err *FfiError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err FfiError) Error() string {
	return fmt.Sprintf("FfiError: %s", err.err.Error())
}

func (err FfiError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrFfiErrorWalletError = fmt.Errorf("FfiErrorWalletError")
var ErrFfiErrorInvalidInput = fmt.Errorf("FfiErrorInvalidInput")
var ErrFfiErrorNetworkError = fmt.Errorf("FfiErrorNetworkError")
var ErrFfiErrorInternalError = fmt.Errorf("FfiErrorInternalError")

// Variant structs
type FfiErrorWalletError struct {
	Msg string
}

func NewFfiErrorWalletError(
	msg string,
) *FfiError {
	return &FfiError{err: &FfiErrorWalletError{
		Msg: msg}}
}

func (e FfiErrorWalletError) destroy() {
	FfiDestroyerString{}.Destroy(e.Msg)
}

func (err FfiErrorWalletError) Error() string {
	return fmt.Sprint("WalletError",
		": ",

		"Msg=",
		err.Msg,
	)
}

func (self FfiErrorWalletError) Is(target error) bool {
	return target == ErrFfiErrorWalletError
}

type FfiErrorInvalidInput struct {
	Msg string
}

func NewFfiErrorInvalidInput(
	msg string,
) *FfiError {
	return &FfiError{err: &FfiErrorInvalidInput{
		Msg: msg}}
}

func (e FfiErrorInvalidInput) destroy() {
	FfiDestroyerString{}.Destroy(e.Msg)
}

func (err FfiErrorInvalidInput) Error() string {
	return fmt.Sprint("InvalidInput",
		": ",

		"Msg=",
		err.Msg,
	)
}

func (self FfiErrorInvalidInput) Is(target error) bool {
	return target == ErrFfiErrorInvalidInput
}

type FfiErrorNetworkError struct {
	Msg string
}

func NewFfiErrorNetworkError(
	msg string,
) *FfiError {
	return &FfiError{err: &FfiErrorNetworkError{
		Msg: msg}}
}

func (e FfiErrorNetworkError) destroy() {
	FfiDestroyerString{}.Destroy(e.Msg)
}

func (err FfiErrorNetworkError) Error() string {
	return fmt.Sprint("NetworkError",
		": ",

		"Msg=",
		err.Msg,
	)
}

func (self FfiErrorNetworkError) Is(target error) bool {
	return target == ErrFfiErrorNetworkError
}

type FfiErrorInternalError struct {
	Msg string
}

func NewFfiErrorInternalError(
	msg string,
) *FfiError {
	return &FfiError{err: &FfiErrorInternalError{
		Msg: msg}}
}

func (e FfiErrorInternalError) destroy() {
	FfiDestroyerString{}.Destroy(e.Msg)
}

func (err FfiErrorInternalError) Error() string {
	return fmt.Sprint("InternalError",
		": ",

		"Msg=",
		err.Msg,
	)
}

func (self FfiErrorInternalError) Is(target error) bool {
	return target == ErrFfiErrorInternalError
}

type FfiConverterFfiError struct{}

var FfiConverterFfiErrorINSTANCE = FfiConverterFfiError{}

func (c FfiConverterFfiError) Lift(eb RustBufferI) *FfiError {
	return LiftFromRustBuffer[*FfiError](c, eb)
}

func (c FfiConverterFfiError) Lower(value *FfiError) C.RustBuffer {
	return LowerIntoRustBuffer[*FfiError](c, value)
}

func (c FfiConverterFfiError) Read(reader io.Reader) *FfiError {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &FfiError{&FfiErrorWalletError{
			Msg: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 2:
		return &FfiError{&FfiErrorInvalidInput{
			Msg: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 3:
		return &FfiError{&FfiErrorNetworkError{
			Msg: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 4:
		return &FfiError{&FfiErrorInternalError{
			Msg: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterFfiError.Read()", errorID))
	}
}

func (c FfiConverterFfiError) Write(writer io.Writer, value *FfiError) {
	switch variantValue := value.err.(type) {
	case *FfiErrorWalletError:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Msg)
	case *FfiErrorInvalidInput:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Msg)
	case *FfiErrorNetworkError:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Msg)
	case *FfiErrorInternalError:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Msg)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterFfiError.Write", value))
	}
}

type FfiDestroyerFfiError struct{}

func (_ FfiDestroyerFfiError) Destroy(value *FfiError) {
	switch variantValue := value.err.(type) {
	case FfiErrorWalletError:
		variantValue.destroy()
	case FfiErrorInvalidInput:
		variantValue.destroy()
	case FfiErrorNetworkError:
		variantValue.destroy()
	case FfiErrorInternalError:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerFfiError.Destroy", value))
	}
}

type FfiMintQuoteState uint

const (
	FfiMintQuoteStateUnpaid FfiMintQuoteState = 1
	FfiMintQuoteStatePaid   FfiMintQuoteState = 2
	FfiMintQuoteStateIssued FfiMintQuoteState = 3
)

type FfiConverterFfiMintQuoteState struct{}

var FfiConverterFfiMintQuoteStateINSTANCE = FfiConverterFfiMintQuoteState{}

func (c FfiConverterFfiMintQuoteState) Lift(rb RustBufferI) FfiMintQuoteState {
	return LiftFromRustBuffer[FfiMintQuoteState](c, rb)
}

func (c FfiConverterFfiMintQuoteState) Lower(value FfiMintQuoteState) C.RustBuffer {
	return LowerIntoRustBuffer[FfiMintQuoteState](c, value)
}
func (FfiConverterFfiMintQuoteState) Read(reader io.Reader) FfiMintQuoteState {
	id := readInt32(reader)
	return FfiMintQuoteState(id)
}

func (FfiConverterFfiMintQuoteState) Write(writer io.Writer, value FfiMintQuoteState) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerFfiMintQuoteState struct{}

func (_ FfiDestroyerFfiMintQuoteState) Destroy(value FfiMintQuoteState) {
}

type FfiSendKind interface {
	Destroy()
}
type FfiSendKindOnlineExact struct {
}

func (e FfiSendKindOnlineExact) Destroy() {
}

type FfiSendKindOnlineTolerance struct {
	Tolerance FfiAmount
}

func (e FfiSendKindOnlineTolerance) Destroy() {
	FfiDestroyerFfiAmount{}.Destroy(e.Tolerance)
}

type FfiSendKindOfflineExact struct {
}

func (e FfiSendKindOfflineExact) Destroy() {
}

type FfiSendKindOfflineTolerance struct {
	Tolerance FfiAmount
}

func (e FfiSendKindOfflineTolerance) Destroy() {
	FfiDestroyerFfiAmount{}.Destroy(e.Tolerance)
}

type FfiConverterFfiSendKind struct{}

var FfiConverterFfiSendKindINSTANCE = FfiConverterFfiSendKind{}

func (c FfiConverterFfiSendKind) Lift(rb RustBufferI) FfiSendKind {
	return LiftFromRustBuffer[FfiSendKind](c, rb)
}

func (c FfiConverterFfiSendKind) Lower(value FfiSendKind) C.RustBuffer {
	return LowerIntoRustBuffer[FfiSendKind](c, value)
}
func (FfiConverterFfiSendKind) Read(reader io.Reader) FfiSendKind {
	id := readInt32(reader)
	switch id {
	case 1:
		return FfiSendKindOnlineExact{}
	case 2:
		return FfiSendKindOnlineTolerance{
			FfiConverterFfiAmountINSTANCE.Read(reader),
		}
	case 3:
		return FfiSendKindOfflineExact{}
	case 4:
		return FfiSendKindOfflineTolerance{
			FfiConverterFfiAmountINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterFfiSendKind.Read()", id))
	}
}

func (FfiConverterFfiSendKind) Write(writer io.Writer, value FfiSendKind) {
	switch variant_value := value.(type) {
	case FfiSendKindOnlineExact:
		writeInt32(writer, 1)
	case FfiSendKindOnlineTolerance:
		writeInt32(writer, 2)
		FfiConverterFfiAmountINSTANCE.Write(writer, variant_value.Tolerance)
	case FfiSendKindOfflineExact:
		writeInt32(writer, 3)
	case FfiSendKindOfflineTolerance:
		writeInt32(writer, 4)
		FfiConverterFfiAmountINSTANCE.Write(writer, variant_value.Tolerance)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterFfiSendKind.Write", value))
	}
}

type FfiDestroyerFfiSendKind struct{}

func (_ FfiDestroyerFfiSendKind) Destroy(value FfiSendKind) {
	value.Destroy()
}

type FfiSplitTarget uint

const (
	FfiSplitTargetNone    FfiSplitTarget = 1
	FfiSplitTargetDefault FfiSplitTarget = 2
)

type FfiConverterFfiSplitTarget struct{}

var FfiConverterFfiSplitTargetINSTANCE = FfiConverterFfiSplitTarget{}

func (c FfiConverterFfiSplitTarget) Lift(rb RustBufferI) FfiSplitTarget {
	return LiftFromRustBuffer[FfiSplitTarget](c, rb)
}

func (c FfiConverterFfiSplitTarget) Lower(value FfiSplitTarget) C.RustBuffer {
	return LowerIntoRustBuffer[FfiSplitTarget](c, value)
}
func (FfiConverterFfiSplitTarget) Read(reader io.Reader) FfiSplitTarget {
	id := readInt32(reader)
	return FfiSplitTarget(id)
}

func (FfiConverterFfiSplitTarget) Write(writer io.Writer, value FfiSplitTarget) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerFfiSplitTarget struct{}

func (_ FfiDestroyerFfiSplitTarget) Destroy(value FfiSplitTarget) {
}

type FfiConverterOptionalUint64 struct{}

var FfiConverterOptionalUint64INSTANCE = FfiConverterOptionalUint64{}

func (c FfiConverterOptionalUint64) Lift(rb RustBufferI) *uint64 {
	return LiftFromRustBuffer[*uint64](c, rb)
}

func (_ FfiConverterOptionalUint64) Read(reader io.Reader) *uint64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint64) Lower(value *uint64) C.RustBuffer {
	return LowerIntoRustBuffer[*uint64](c, value)
}

func (_ FfiConverterOptionalUint64) Write(writer io.Writer, value *uint64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint64 struct{}

func (_ FfiDestroyerOptionalUint64) Destroy(value *uint64) {
	if value != nil {
		FfiDestroyerUint64{}.Destroy(*value)
	}
}

type FfiConverterOptionalString struct{}

var FfiConverterOptionalStringINSTANCE = FfiConverterOptionalString{}

func (c FfiConverterOptionalString) Lift(rb RustBufferI) *string {
	return LiftFromRustBuffer[*string](c, rb)
}

func (_ FfiConverterOptionalString) Read(reader io.Reader) *string {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterStringINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalString) Lower(value *string) C.RustBuffer {
	return LowerIntoRustBuffer[*string](c, value)
}

func (_ FfiConverterOptionalString) Write(writer io.Writer, value *string) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalString struct{}

func (_ FfiDestroyerOptionalString) Destroy(value *string) {
	if value != nil {
		FfiDestroyerString{}.Destroy(*value)
	}
}

type FfiConverterOptionalFfiSendMemo struct{}

var FfiConverterOptionalFfiSendMemoINSTANCE = FfiConverterOptionalFfiSendMemo{}

func (c FfiConverterOptionalFfiSendMemo) Lift(rb RustBufferI) *FfiSendMemo {
	return LiftFromRustBuffer[*FfiSendMemo](c, rb)
}

func (_ FfiConverterOptionalFfiSendMemo) Read(reader io.Reader) *FfiSendMemo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterFfiSendMemoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalFfiSendMemo) Lower(value *FfiSendMemo) C.RustBuffer {
	return LowerIntoRustBuffer[*FfiSendMemo](c, value)
}

func (_ FfiConverterOptionalFfiSendMemo) Write(writer io.Writer, value *FfiSendMemo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterFfiSendMemoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalFfiSendMemo struct{}

func (_ FfiDestroyerOptionalFfiSendMemo) Destroy(value *FfiSendMemo) {
	if value != nil {
		FfiDestroyerFfiSendMemo{}.Destroy(*value)
	}
}

type FfiConverterMapStringString struct{}

var FfiConverterMapStringStringINSTANCE = FfiConverterMapStringString{}

func (c FfiConverterMapStringString) Lift(rb RustBufferI) map[string]string {
	return LiftFromRustBuffer[map[string]string](c, rb)
}

func (_ FfiConverterMapStringString) Read(reader io.Reader) map[string]string {
	result := make(map[string]string)
	length := readInt32(reader)
	for i := int32(0); i < length; i++ {
		key := FfiConverterStringINSTANCE.Read(reader)
		value := FfiConverterStringINSTANCE.Read(reader)
		result[key] = value
	}
	return result
}

func (c FfiConverterMapStringString) Lower(value map[string]string) C.RustBuffer {
	return LowerIntoRustBuffer[map[string]string](c, value)
}

func (_ FfiConverterMapStringString) Write(writer io.Writer, mapValue map[string]string) {
	if len(mapValue) > math.MaxInt32 {
		panic("map[string]string is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(mapValue)))
	for key, value := range mapValue {
		FfiConverterStringINSTANCE.Write(writer, key)
		FfiConverterStringINSTANCE.Write(writer, value)
	}
}

type FfiDestroyerMapStringString struct{}

func (_ FfiDestroyerMapStringString) Destroy(mapValue map[string]string) {
	for key, value := range mapValue {
		FfiDestroyerString{}.Destroy(key)
		FfiDestroyerString{}.Destroy(value)
	}
}

// Generate a 12-word mnemonic phrase
func GenerateMnemonic() (string, error) {
	_uniffiRV, _uniffiErr := rustCallWithError[FfiError](FfiConverterFfiError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_cdk_ffi_fn_func_generate_mnemonic(_uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), nil
	}
}
