package plugin

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"slices"
	"sync"
	"unsafe"

	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/sourcemap"
)

type ServiceCodeError struct {
	Message string
	Start   int
	End     int
}

type ServiceCode struct {
	Errors              []ServiceCodeError
	ServiceText         []byte
	ScriptKind          ScriptKind
	Mappings            []Mapping
	IgnoreMappings      []IgnoreDirectiveMapping
	ExpectErrorMappings []ExpectErrorDirectiveMapping
	DeclarationFile     bool
}

type PluginInstance struct {
	CreateServiceCode func(cwd, configFileName, fileName string, sourceText string) *ServiceCode
}

type PluginOptions struct {
	Input  io.Reader
	Output io.Writer
	// Example: [".vue"]
	ExtraExtensions []string
	Setup           func() PluginInstance
}

func ensureCap(b []byte, needed uint32) []byte {
	if b == nil || uint32(cap(b)) < needed {
		b = make([]byte, needed)
	}
	return b[:needed]
}

type ScriptKind uint8

const (
	ScriptKindJS ScriptKind = iota
	ScriptKindJSX
	ScriptKindTS
	ScriptKindTSX
)

type MsgKind uint8

const (
	MsgKindCreateServiceCode MsgKind = iota
	MsgKindCreateServiceCodeResponse
)

type ServiceCodeProperties uint8

const (
	ServiceCodePropertiesError ServiceCodeProperties = 1 << iota
	ServiceCodePropertiesDeclarationFile
)

type Mapping struct {
	SourceOffset  uint32
	ServiceOffset uint32
	SourceLength  uint32
	ServiceLength uint32
}

type IgnoreDirectiveMapping struct {
	ServiceOffset uint32
	ServiceLength uint32
}

type ExpectErrorDirectiveMapping struct {
	SourceOffset  uint32
	ServiceOffset uint32
	SourceLength  uint32
	ServiceLength uint32
}

type InitializationMessage struct {
	ExtraExtensions []string `json:"extraExtensions"`
}

func Run(opts PluginOptions) {
	var header [5]byte
	var recvBuf []byte

	{
		if opts.ExtraExtensions == nil {
			opts.ExtraExtensions = []string{}
		}
		initialization, err := json.Marshal(InitializationMessage{
			ExtraExtensions: opts.ExtraExtensions,
		})
		if err != nil {
			panic(err)
		}

		binary.LittleEndian.PutUint32(header[:], uint32(len(initialization)))
		if _, err = opts.Output.Write(header[:4]); err != nil {
			panic(err)
		}
		if _, err = opts.Output.Write(initialization); err != nil {
			panic(err)
		}
	}

	tasks := make(chan []byte, 1000)
	var writeMu sync.Mutex
	for range 4 {
		instance := opts.Setup()
		go func() {
			var sendBuf []byte
			for task := range tasks {
				offset := uint32(0)
				reqId := binary.LittleEndian.Uint64(task[offset:])
				offset += 8
				cwdLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				cwd := string(task[offset : offset+cwdLen])
				offset += cwdLen
				configFileNameLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				configFileName := string(task[offset : offset+configFileNameLen])
				offset += configFileNameLen
				fileNameLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				fileName := string(task[offset : offset+fileNameLen])
				offset += fileNameLen
				sourceTextLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				sourceText := string(task[offset : offset+sourceTextLen])
				offset += sourceTextLen

				if configFileName == "/dev/null/inferred" {
					configFileName = ""
				}
				res := instance.CreateServiceCode(cwd, configFileName, fileName, sourceText)

				var properties ServiceCodeProperties

				if len(res.Errors) > 0 {
					properties |= ServiceCodePropertiesError
					errorsLen := 0
					for _, err := range res.Errors {
						errorsLen += 4 + len(err.Message) + 4 + 4
					}
					responsePayloadLen := uint32(8 + 1 + 4 + errorsLen)
					offset = 0
					sendBuf = ensureCap(sendBuf, 5+responsePayloadLen)
					sendBuf[0] = byte(MsgKindCreateServiceCodeResponse)
					offset += 1
					binary.LittleEndian.PutUint32(sendBuf[offset:], responsePayloadLen)
					offset += 4
					binary.LittleEndian.PutUint64(sendBuf[offset:], reqId)
					offset += 8
					sendBuf[offset] = byte(properties)
					offset += 1
					binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.Errors)))
					offset += 4
					for _, err := range res.Errors {
						binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(err.Message)))
						offset += 4
						copy(sendBuf[offset:], err.Message)
						offset += uint32(len(err.Message))
						binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(err.Start))
						offset += 4
						binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(err.End))
						offset += 4
					}
				} else {
					mappingsByteLen := len(res.Mappings) * int(unsafe.Sizeof(Mapping{}))
					ignoreMappingsByteLen := len(res.IgnoreMappings) * int(unsafe.Sizeof(IgnoreDirectiveMapping{}))
					expectErrorMappingsByteLen := len(res.ExpectErrorMappings) * int(unsafe.Sizeof(ExpectErrorDirectiveMapping{}))
					responsePayloadLen := uint32(8 + 1 + 1 + 4 + len(res.ServiceText) + 4 + mappingsByteLen + 4 + ignoreMappingsByteLen + 4 + expectErrorMappingsByteLen)

					offset = 0
					sendBuf = ensureCap(sendBuf, 5+responsePayloadLen)
					sendBuf[0] = byte(MsgKindCreateServiceCodeResponse)
					offset += 1
					binary.LittleEndian.PutUint32(sendBuf[offset:], responsePayloadLen)
					offset += 4
					binary.LittleEndian.PutUint64(sendBuf[offset:], reqId)
					offset += 8
					if res.DeclarationFile {
						properties |= ServiceCodePropertiesDeclarationFile
					}
					sendBuf[offset] = byte(properties)
					offset += 1
					sendBuf[offset] = byte(res.ScriptKind)
					offset += 1
					binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.ServiceText)))
					offset += 4
					copy(sendBuf[offset:], res.ServiceText)
					offset += uint32(len(res.ServiceText))
					binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.Mappings)))
					offset += 4
					if len(res.Mappings) > 0 {
						byteLen := len(res.Mappings) * int(unsafe.Sizeof(Mapping{}))
						bytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(res.Mappings))), byteLen)
						copy(sendBuf[offset:], bytes)
						offset += uint32(len(bytes))
					}
					binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.IgnoreMappings)))
					offset += 4
					if len(res.IgnoreMappings) > 0 {
						byteLen := len(res.IgnoreMappings) * int(unsafe.Sizeof(IgnoreDirectiveMapping{}))
						bytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(res.IgnoreMappings))), byteLen)
						copy(sendBuf[offset:], bytes)
						offset += uint32(len(bytes))
					}
					binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.ExpectErrorMappings)))
					offset += 4
					if len(res.ExpectErrorMappings) > 0 {
						byteLen := len(res.ExpectErrorMappings) * int(unsafe.Sizeof(ExpectErrorDirectiveMapping{}))
						bytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(res.ExpectErrorMappings))), byteLen)
						copy(sendBuf[offset:], bytes)
						offset += uint32(len(bytes))
					}
				}

				writeMu.Lock()
				_, err := opts.Output.Write(sendBuf)
				writeMu.Unlock()
				if err != nil {
					panic(err)
				}
			}
		}()
	}
	for {
		_, err := io.ReadFull(opts.Input, header[:])
		if err == io.EOF {
			return
		} else if err != nil {
			panic(err)
		}

		msgKind := MsgKind(header[0])
		payloadLen := binary.LittleEndian.Uint32(header[1:])
		recvBuf = ensureCap(recvBuf, payloadLen)

		if _, err := io.ReadFull(opts.Input, recvBuf); err != nil {
			panic(err)
		}

		switch msgKind {
		case MsgKindCreateServiceCode:
			tasks <- slices.Clone(recvBuf[:payloadLen])
		}
	}
}

func SourceMapToMappings(sourceText string, serviceText string, sourceMap string) []Mapping {
	dec := sourcemap.DecodeMappings(sourceMap)
	serviceLineMap := core.ComputeECMALineStarts(serviceText)
	sourceLineMap := core.ComputeECMALineStarts(sourceText)
	mappings := make([]Mapping, 0)

	type currentMapping struct {
		genOffset    uint32
		sourceOffset uint32
	}

	var current *currentMapping

	for decoded, done := dec.Next(); !done; decoded, done = dec.Next() {
		if decoded == nil {
			continue
		}
		genOffset := uint32(scanner.ComputePositionOfLineAndCharacterEx(
			serviceLineMap,
			decoded.GeneratedLine,
			decoded.GeneratedCharacter,
			&serviceText,
			false,
		))
		if current != nil {
			length := genOffset - current.genOffset
			if length > 0 {
				sourceEnd := min(current.sourceOffset+length, uint32(len(sourceText)))
				genEnd := min(current.genOffset+length, uint32(len(serviceText)))
				sourceChunk := sourceText[current.sourceOffset:sourceEnd]
				genChunk := serviceText[current.genOffset:genEnd]
				if sourceChunk != genChunk {
					length = 0
					maxLen := min(len(sourceChunk), len(genChunk))
					for i := range maxLen {
						if sourceChunk[i] == genChunk[i] {
							length = uint32(i) + 1
						} else {
							break
						}
					}
				}
			}
			if length > 0 {
				if len(mappings) > 0 {
					last := &mappings[len(mappings)-1]
					if last.ServiceOffset+last.SourceLength == current.genOffset &&
						last.SourceOffset+last.SourceLength == current.sourceOffset {
						last.SourceLength += length
					} else {
						mappings = append(mappings, Mapping{
							SourceOffset:  current.sourceOffset,
							ServiceOffset: current.genOffset,
							SourceLength:  length,
						})
					}
				} else {
					mappings = append(mappings, Mapping{
						SourceOffset:  current.sourceOffset,
						ServiceOffset: current.genOffset,
						SourceLength:  length,
					})
				}
			}
			current = nil
		}
		if decoded.IsSourceMapping() {
			if decoded.SourceIndex != 0 {
				continue
			}
			sourceOffset := uint32(scanner.ComputePositionOfLineAndCharacterEx(
				sourceLineMap,
				decoded.SourceLine,
				decoded.SourceCharacter,
				&sourceText,
				true,
			))
			current = &currentMapping{
				genOffset:    genOffset,
				sourceOffset: sourceOffset,
			}
		}
	}

	if err := dec.Error(); err != nil {
		panic(err)
	}

	return mappings
}
