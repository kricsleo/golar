package tscodegenplugin

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/auvred/golar/plugin"
	"github.com/auvred/golar/util"
)

var ipcDebug = util.NewDebug("tscodegenplugin:ipc")
var ipcDebugVerbose = util.NewDebug("tscodegenplugin:ipc:verbose")

type IpcPlugin struct {
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	sendBuf []byte
	mu      sync.Mutex

	reqId                     atomic.Uint64
	createServiceCodeRequests sync.Map

	extensions []plugin.FileExtension
}

var _ Plugin = (*IpcPlugin)(nil)

type serviceCodeRequest struct {
	started  time.Time
	fileName string
	callback func(payload []byte)
}

func NewIpcPlugin(args []string, extensions []plugin.FileExtension) (*IpcPlugin, error) {
	t := time.Now()
	p := &IpcPlugin{extensions: extensions}
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
	err = cmd.Start()
	ipcDebug.Printf("started %#v plugin; err: %#v; +%v", args, err, time.Since(t))
	if err != nil {
		return nil, err
	}
	var header [5]byte
	var recvBuf []byte

	go func() {
		for {
			_, err := io.ReadFull(p.stdout, header[:])
			if err != nil {
				if err == io.EOF {
					// TODO?
					panic(fmt.Sprintf("plugin %#v exited", args))
				}
				panic(err)
			}
			msgKind := plugin.MsgKind(header[0])
			payloadLen := binary.LittleEndian.Uint32(header[1:])
			recvBuf = ensureCap(recvBuf, payloadLen)
			if _, err := io.ReadFull(p.stdout, recvBuf); err != nil {
				panic(err)
			}
			switch msgKind {
			case plugin.MsgKindCreateServiceCodeResponse:
				reqId := binary.LittleEndian.Uint64(recvBuf)
				f, _ := p.createServiceCodeRequests.LoadAndDelete(reqId)
				req := f.(serviceCodeRequest)
				ipcDebug.Printf("createServiceCode(%v) +%v", req.fileName, time.Since(req.started))
				req.callback(recvBuf[8:])
			}
		}
	}()

	return p, nil
}

func (p *IpcPlugin) Extensions() []plugin.FileExtension {
	return p.extensions
}

func (p *IpcPlugin) sendMessage(msgKind plugin.MsgKind, payload []byte) error {
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

func (p *IpcPlugin) CreateServiceCode(req CreateServiceCodeRequest) CreateServiceCodeResponse {
	var response CreateServiceCodeResponse
	var wg sync.WaitGroup
	wg.Add(1)

	reqId := p.reqId.Add(1)

	p.createServiceCodeRequests.Store(reqId, serviceCodeRequest{
		started:  time.Now(),
		fileName: req.FileName,
		callback: func(payload []byte) {
			defer wg.Done()

			response = decodeCreateServiceCodeResponse(payload)

			ipcDebugVerbose.Printf("createServiceCode(%v) returned %#v", req.FileName, response)
		},
	})
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sendBuf = ensureCap(p.sendBuf, uint32(8+encodedLenCreateServiceCodeRequest(req)))
	offset := 0
	binary.LittleEndian.PutUint64(p.sendBuf, reqId)
	offset += 8
	encodeCreateServiceCodeRequest(p.sendBuf[offset:], req)

	p.sendMessage(plugin.MsgKindCreateServiceCode, p.sendBuf)

	wg.Wait()
	return response
}
