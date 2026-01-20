package pluginhost

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/plugin"
)

type Plugin struct {
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	sendBuf []byte
	mu sync.Mutex

	reqId atomic.Uint64
	createServiceCodeRequests sync.Map
}

func NewPlugin(args []string) (*Plugin, error) {
	p := Plugin{}
	cmd := exec.Command(args[0], args[1:]...)
	var err error
	p.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %v", err)
	}
	p.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		var header [5]byte
		var recvBuf []byte
		for {
			_, err := io.ReadFull(p.stdout, header[:])
			if err != nil {
				// TODO?
				// if err == io.EOF {
				// 	return
				// }
				panic(err)
			}
			msgKind := plugin.MsgKind(header[0])
			payloadLen := binary.LittleEndian.Uint32(header[1:])
			recvBuf = ensureCap(recvBuf, payloadLen)
			if _, err := io.ReadFull(p.stdout, recvBuf); err != nil {
				panic(err)
			}
			switch msgKind {
			case plugin.MsgKindInitializeResponse:
			case plugin.MsgKindCreateServiceCodeResponse:
				reqId := binary.LittleEndian.Uint64(recvBuf)
				f, _ := p.createServiceCodeRequests.LoadAndDelete(reqId)
				f.(func (r[]byte))(recvBuf[8:])
			}
		}
	}()

	return &p, nil
}

func (p *Plugin) sendMessage(msgKind plugin.MsgKind, payload []byte) error {
	var header [5]byte
	header[0] = byte(msgKind)
	binary.LittleEndian.PutUint32(header[1:], uint32(len(payload)))
	_, err := p.stdin.Write(header[:])
	if err != nil {
		return err
	}
	_, err = p.stdin.Write(payload)
	return err
}

func ensureCap(b []byte, needed uint32) []byte {
	if b == nil || uint32(cap(b)) < needed {
		b = make([]byte, needed)
	}
	return b[:needed]
}

type CreateServiceCodeResponse struct {
	ServiceText string
	SourceMap string
	Mappings []mapping.Mapping
	IgnoreMappings []mapping.IgnoreDirectiveMapping
}

func (p *Plugin) CreateServiceCode(fileName string, sourceText string) <- chan CreateServiceCodeResponse {
	ch := make(chan CreateServiceCodeResponse, 1)

	reqId := p.reqId.Add(1)
	p.createServiceCodeRequests.Store(reqId, func (payload []byte) {
		offset := uint32(0)

		properties := payload[offset]
		offset += 1

		response := CreateServiceCodeResponse{}

		if properties & 1 != 0 {
			serviceTextLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.ServiceText = string(payload[offset:offset+serviceTextLen])
			offset += serviceTextLen

			sourceMapLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.SourceMap = string(payload[offset:offset+sourceMapLen])
			offset += sourceMapLen
		} else {
			serviceTextLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.ServiceText = string(payload[offset:offset+serviceTextLen])
			offset += serviceTextLen

			mappingsCount := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.Mappings = make([]mapping.Mapping, mappingsCount)
			for i := range mappingsCount {
				hasServiceLengths := (payload[offset] & (1 << 7)) != 0
				count := payload[offset] & ^byte(1 << 7)
				offset += 1

				response.Mappings[i].SourceOffsets = make([]int, count)
				for j := range count {
					response.Mappings[i].SourceOffsets[j] = int(binary.LittleEndian.Uint32(payload[offset:]))
					offset += 4
				}

				response.Mappings[i].ServiceOffsets = make([]int, count)
				for j := range count {
					response.Mappings[i].ServiceOffsets[j] = int(binary.LittleEndian.Uint32(payload[offset:]))
					offset += 4
				}

				response.Mappings[i].SourceLengths = make([]int, count)
				for j := range count {
					response.Mappings[i].SourceLengths[j] = int(binary.LittleEndian.Uint32(payload[offset:]))
					offset += 4
				}

				if hasServiceLengths {
					response.Mappings[i].ServiceLengths = make([]int, count)
					for j := range count {
						response.Mappings[i].ServiceLengths[j] = int(binary.LittleEndian.Uint32(payload[offset:]))
						offset += 4
					}
				}
			}

			ignoreMappingsCount := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.IgnoreMappings = make([]mapping.IgnoreDirectiveMapping, ignoreMappingsCount)
			for i := range ignoreMappingsCount {
				response.IgnoreMappings[i].ServiceOffset = int(binary.LittleEndian.Uint32(payload[offset:]))
				offset += 4
				response.IgnoreMappings[i].ServiceLength = int(binary.LittleEndian.Uint32(payload[offset:]))
				offset += 4
			}
		}

		ch <- response
	})
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sendBuf = ensureCap(p.sendBuf, uint32(8 + 4 + len(fileName) + 4 + len(sourceText)))
	binary.LittleEndian.PutUint64(p.sendBuf, reqId)
	offset := 8
	binary.LittleEndian.PutUint32(p.sendBuf[offset:], uint32(len(fileName)))
	offset += 4
	copy(p.sendBuf[offset:], fileName)
	offset += len(fileName)
	binary.LittleEndian.PutUint32(p.sendBuf[offset:], uint32(len(sourceText)))
	offset += 4
	copy(p.sendBuf[offset:], sourceText)
	offset += len(sourceText)
	p.sendMessage(plugin.MsgKindCreateServiceCode, p.sendBuf)

	return ch
}
