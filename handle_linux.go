package netlink

import (
	"fmt"
	"time"

	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

// Empty handle used by the netlink package methods
//
// NetLink包方法使用的空句柄  ，bridge用的这个句柄
var pkgHandle = &Handle{}

// Handle is an handle for the netlink requests on a
// specific network namespace. All the requests on the
// same netlink family share the same netlink socket,
// which gets released when the handle is deleted.
//
// 句柄是特定网络命名空间上的NetLink请求的句柄。
// 同一NetLink系列上的所有请求共享相同的NetLink套接字，删除句柄时会释放该套接字。
//
type Handle struct {
	sockets      map[int]*nl.SocketHandle
	lookupByDump bool
}

// SupportsNetlinkFamily reports whether the passed netlink family is supported by this Handle
//
// SupportsNetlinkFamily报告此句柄是否支持传递的NetLink系列
//
func (h *Handle) SupportsNetlinkFamily(nlFamily int) bool {
	_, ok := h.sockets[nlFamily]
	return ok
}

// NewHandle returns a netlink handle on the current network namespace.
// Caller may specify the netlink families the handle should support.
// If no families are specified, all the families the netlink package
// supports will be automatically added.
//
// NewHandle返回当前网络命名空间上的NetLink句柄。
// 调用者可以指定句柄应该支持的NetLink系列。
// 如果未指定族，将自动添加NetLink软件包支持的所有族。
//
func NewHandle(nlFamilies ...int) (*Handle, error) {
	return newHandle(netns.None(), netns.None(), nlFamilies...)
}

// SetSocketTimeout sets the send and receive timeout for each socket in the
// netlink handle. Although the socket timeout has granularity of one
// microsecond, the effective granularity is floored by the kernel timer tick,
// which default value is four milliseconds.
//
// SetSocketTimeout设置NetLink句柄中每个套接字的发送和接收超时。
// 尽管套接字超时的粒度为1微秒，但有效粒度会被内核计时器计时器计时(默认值为4毫秒)所破坏。
//
func (h *Handle) SetSocketTimeout(to time.Duration) error {
	if to < time.Microsecond {
		return fmt.Errorf("invalid timeout, minimul value is %s", time.Microsecond)
	}
	tv := unix.NsecToTimeval(to.Nanoseconds())
	for _, sh := range h.sockets {
		if err := sh.Socket.SetSendTimeout(&tv); err != nil {
			return err
		}
		if err := sh.Socket.SetReceiveTimeout(&tv); err != nil {
			return err
		}
	}
	return nil
}

// SetSocketReceiveBufferSize sets the receive buffer size for each
// socket in the netlink handle. The maximum value is capped by
// /proc/sys/net/core/rmem_max.
func (h *Handle) SetSocketReceiveBufferSize(size int, force bool) error {
	opt := unix.SO_RCVBUF
	if force {
		opt = unix.SO_RCVBUFFORCE
	}
	for _, sh := range h.sockets {
		fd := sh.Socket.GetFd()
		err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, opt, size)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetSocketReceiveBufferSize gets the receiver buffer size for each
// socket in the netlink handle. The retrieved value should be the
// double to the one set for SetSocketReceiveBufferSize.
func (h *Handle) GetSocketReceiveBufferSize() ([]int, error) {
	results := make([]int, len(h.sockets))
	i := 0
	for _, sh := range h.sockets {
		fd := sh.Socket.GetFd()
		size, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
		if err != nil {
			return nil, err
		}
		results[i] = size
		i++
	}
	return results, nil
}

// NewHandleAt returns a netlink handle on the network namespace
// specified by ns. If ns=netns.None(), current network namespace
// will be assumed
func NewHandleAt(ns netns.NsHandle, nlFamilies ...int) (*Handle, error) {
	return newHandle(ns, netns.None(), nlFamilies...)
}

// NewHandleAtFrom works as NewHandle but allows client to specify the
// new and the origin netns Handle.
func NewHandleAtFrom(newNs, curNs netns.NsHandle) (*Handle, error) {
	return newHandle(newNs, curNs)
}

func newHandle(newNs, curNs netns.NsHandle, nlFamilies ...int) (*Handle, error) {
	h := &Handle{sockets: map[int]*nl.SocketHandle{}}
	fams := nl.SupportedNlFamilies
	if len(nlFamilies) != 0 {
		fams = nlFamilies
	}
	for _, f := range fams {
		s, err := nl.GetNetlinkSocketAt(newNs, curNs, f)
		if err != nil {
			return nil, err
		}
		h.sockets[f] = &nl.SocketHandle{Socket: s}
	}
	return h, nil
}

// Delete releases the resources allocated to this handle
func (h *Handle) Delete() {
	for _, sh := range h.sockets {
		sh.Close()
	}
	h.sockets = nil
}

// 设置 Netlink消息头，如果有socket则设置socket
func (h *Handle) newNetlinkRequest(proto, flags int) *nl.NetlinkRequest {
	// Do this so that package API still use nl package variable nextSeqNr
	//
	// 这样做可以使包API仍然使用NL包变量nextSeqNr
	if h.sockets == nil {
		// bridge 的h是pkgHandle，sockets是nil
		// 设置Netlink消息头
		return nl.NewNetlinkRequest(proto, flags)
	}
	return &nl.NetlinkRequest{
		NlMsghdr: unix.NlMsghdr{
			Len:   uint32(unix.SizeofNlMsghdr),
			Type:  uint16(proto),
			Flags: unix.NLM_F_REQUEST | uint16(flags),
			// Todo 注意这里没有Seq，why
		},
		Sockets: h.sockets,  // 有socket则在request设置socket
	}
}

/*
func NewNetlinkRequest(proto, flags int) *NetlinkRequest {
	return &NetlinkRequest{
		NlMsghdr: unix.NlMsghdr{
			Len:   uint32(unix.SizeofNlMsghdr),
			Type:  uint16(proto),
			Flags: unix.NLM_F_REQUEST | uint16(flags),
			Seq:   atomic.AddUint32(&nextSeqNr, 1),
		},
	}
}
*/
