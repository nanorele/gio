package driver

import (
	"fmt"
	"unsafe"

	"github.com/nanorele/gio/internal/gl"
)

type API interface {
	implementsAPI()
}

type RenderTarget interface {
	ImplementsRenderTarget()
}

type OpenGLRenderTarget gl.Framebuffer

type Direct3D11RenderTarget struct {
	RenderTarget unsafe.Pointer
}

type MetalRenderTarget struct {
	Texture uintptr
}

type VulkanRenderTarget struct {
	WaitSem uint64

	SignalSem uint64

	Fence uint64

	Image uint64

	Framebuffer uint64
}

type OpenGL struct {
	ES bool

	Context gl.Context

	Shared bool
}

type Direct3D11 struct {
	Device unsafe.Pointer
}

type Metal struct {
	Device uintptr

	Queue uintptr

	PixelFormat int
}

type Vulkan struct {
	PhysDevice unsafe.Pointer

	Device unsafe.Pointer

	QueueFamily int

	QueueIndex int

	Format int
}

var (
	NewOpenGLDevice     func(api OpenGL) (Device, error)
	NewDirect3D11Device func(api Direct3D11) (Device, error)
	NewMetalDevice      func(api Metal) (Device, error)
	NewVulkanDevice     func(api Vulkan) (Device, error)
)

func NewDevice(api API) (Device, error) {
	switch api := api.(type) {
	case OpenGL:
		if NewOpenGLDevice != nil {
			return NewOpenGLDevice(api)
		}
	case Direct3D11:
		if NewDirect3D11Device != nil {
			return NewDirect3D11Device(api)
		}
	case Metal:
		if NewMetalDevice != nil {
			return NewMetalDevice(api)
		}
	case Vulkan:
		if NewVulkanDevice != nil {
			return NewVulkanDevice(api)
		}
	}
	return nil, fmt.Errorf("driver: no driver available for the API %T", api)
}

func (OpenGL) implementsAPI()                          {}
func (Direct3D11) implementsAPI()                      {}
func (Metal) implementsAPI()                           {}
func (Vulkan) implementsAPI()                          {}
func (OpenGLRenderTarget) ImplementsRenderTarget()     {}
func (Direct3D11RenderTarget) ImplementsRenderTarget() {}
func (MetalRenderTarget) ImplementsRenderTarget()      {}
func (VulkanRenderTarget) ImplementsRenderTarget()     {}
